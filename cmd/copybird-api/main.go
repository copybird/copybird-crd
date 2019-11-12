package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/copybird/copybird/core"
	"github.com/iancoleman/strcase"
)

const (
	envMysqlDSN    = "MYSQL_DSN"
	envDumpPath    = "DUMP_PATH"
	envCompression = "COMPRESSION"
	envEncryption  = "ENCRYPTION"
)

func main() {
	mysqlDSN, defined := os.LookupEnv(envMysqlDSN)
	if !defined {
		log.Fatalf("environment variable %q not defined", envMysqlDSN)
	}

	dumpPath, defined := os.LookupEnv(envDumpPath)
	if !defined {
		log.Fatalf("environment variable %q not defined", envDumpPath)
	}

	compression, defined := os.LookupEnv(envCompression)
	if !defined {
		log.Fatalf("environment variable %q not defined", envCompression)
	}

	encryption, defined := os.LookupEnv(envEncryption)
	if !defined {
		log.Fatalf("environment variable %q not defined", envEncryption)
	}

	mInput, err := loadModule(core.ModuleGroupBackup, core.ModuleTypeInput, mysqlDSN)
	if err != nil {
		log.Panic(err)
	}
	mOutput, err := loadModule(core.ModuleGroupBackup, core.ModuleTypeOutput, dumpPath)
	if err != nil {
		log.Panic(err)
	}

	mCompress, err := loadModule(core.ModuleGroupBackup, core.ModuleTypeCompress, compression)
	if err != nil {
		log.Panic(err)
	}

	mEncrypt, err := loadModule(core.ModuleGroupBackup, core.ModuleTypeEncrypt, encryption)
	if err != nil {
		log.Panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	nextReader, nextWriter := io.Pipe()

	go runModule(mInput, nextWriter, nil, &wg)
	_nextReader, _nextWriter := io.Pipe()
	wg.Add(1)
	go runModule(mCompress, _nextWriter, nextReader, &wg)
	nextReader = _nextReader

	_nextReader, _nextWriter = io.Pipe()
	wg.Add(1)
	go runModule(mEncrypt, _nextWriter, nextReader, &wg)
	nextReader = _nextReader

	go runModule(mOutput, nil, nextReader, &wg)

	wg.Wait()

	log.Println("Mysql backup complete")
}

func loadModule(mGroup core.ModuleGroup, mType core.ModuleType, args string) (core.Module, error) {
	name, params := parseArgs(args)
	module := core.GetModule(mGroup, mType, name)
	if module == nil {
		return nil, fmt.Errorf("module %s/%s not found", mType, name)
	}
	config := module.GetConfig()
	loadConfig(config, params)
	log.Printf("module %s/%s config: %+v", mType, name, config)
	if err := module.InitModule(config); err != nil {
		return nil, fmt.Errorf("init module %s/%s err: %s", mType, name, err)
	}
	return module, nil
}

func loadConfig(cfg interface{}, params map[string]string) error {
	cfgValue := reflect.Indirect(reflect.ValueOf(cfg))
	cfgType := cfgValue.Type()

	for pName, pValue := range params {
		for i := 0; i < cfgType.NumField(); i++ {
			fieldValue := cfgValue.Field(i)
			fieldType := cfgType.Field(i)
			if strcase.ToSnake(fieldType.Name) == pName {
				switch fieldType.Type.Kind() {
				case reflect.String:
					fieldValue.SetString(pValue)
				case reflect.Int:
					val, err := strconv.ParseInt(pValue, 10, 63)
					if err != nil {
						return err
					}
					fieldValue.SetInt(val)
				case reflect.Bool:
					val, err := strconv.ParseBool(pValue)
					if err != nil {
						return err
					}
					fieldValue.SetBool(val)
				default:
					return fmt.Errorf("unsupported config param type: %s %s",
						pName,
						fieldType.Type.Kind().String())
				}
			}
		}
	}
	return nil
}

func runModule(module core.Module, writer io.WriteCloser, reader io.ReadCloser, wg *sync.WaitGroup) {
	defer func(t time.Time) {
		if writer != nil {
			writer.Close()
		}
		if reader != nil {
			reader.Close()
		}
		wg.Done()
		if err := recover(); err != nil {
			log.Printf("module %s/%s err: %s", module.GetType(), module.GetName(), err)
		}
		log.Printf("module %s/%s done by %.2fms", module.GetType(), module.GetName(), time.Since(t).Seconds()*1000)
	}(time.Now())
	err := module.InitPipe(writer, reader)
	if err != nil {
		panic(err)
	}
	err = module.Run()
	if err != nil {
		panic(err)
	}
}

func parseArgs(args string) (string, map[string]string) {
	var moduleName string
	var moduleParams = make(map[string]string)

	parts := strings.Split(args, "::")
	moduleName = parts[0]
	if len(parts) > 1 {
		for _, param := range parts[1:] {
			paramParts := strings.Split(param, "=")
			moduleParams[paramParts[0]] = paramParts[1]
		}
	}

	return moduleName, moduleParams
}
