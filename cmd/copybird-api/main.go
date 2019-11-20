package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/copybird/copybird/core"
	"github.com/copybird/copybird/modules/backup/compress/gzip"
	"github.com/copybird/copybird/modules/backup/compress/lz4"
	"github.com/copybird/copybird/modules/backup/encrypt/aesgcm"
	"github.com/copybird/copybird/modules/backup/input/mongodb"
	"github.com/copybird/copybird/modules/backup/input/mysql"
	postgres "github.com/copybird/copybird/modules/backup/input/postgresql"
	"github.com/copybird/copybird/modules/backup/output/gcp"
	"github.com/copybird/copybird/modules/backup/output/http"
	"github.com/copybird/copybird/modules/backup/output/local"
	"github.com/copybird/copybird/modules/backup/output/s3"
	"github.com/copybird/copybird/modules/backup/output/scp"
	"github.com/kelseyhightower/envconfig"
	"golang.org/x/sync/errgroup"
)

const (
	inputEnv    = "COPYBIRD_INPUT"
	outputEnv   = "COPYBIRD_OUTPUT"
	compressEnv = "COPYBIRD_COMPRESS"
	encryptEnv  = "COPYBIRD_ENCRYPT"
)

func init() {
	core.RegisterModule(&mysql.BackupInputMysql{})
	core.RegisterModule(&postgres.BackupInputPostgresql{})
	core.RegisterModule(&mongodb.BackupInputMongodb{})
	core.RegisterModule(&gzip.BackupCompressGzip{})
	core.RegisterModule(&lz4.BackupCompressLz4{})
	core.RegisterModule(&aesgcm.BackupEncryptAesgcm{})
	core.RegisterModule(&gcp.BackupOutputGcp{})
	core.RegisterModule(&http.BackupOutputHttp{})
	core.RegisterModule(&local.BackupOutputLocal{})
	core.RegisterModule(&s3.BackupOutputS3{})
	core.RegisterModule(&scp.BackupOutputScp{})
}

func main() {
	input, defined := os.LookupEnv(inputEnv)
	if !defined {
		log.Fatalf("environment variable %q not defined", inputEnv)
	}

	output, defined := os.LookupEnv(outputEnv)
	if !defined {
		log.Fatalf("environment variable %q not defined", outputEnv)
	}

	mInput, err := loadModule(core.ModuleGroupBackup, core.ModuleTypeInput, input, inputEnv)
	if err != nil {
		log.Panic(err)
	}
	mOutput, err := loadModule(core.ModuleGroupBackup, core.ModuleTypeOutput, output, outputEnv)
	if err != nil {
		log.Panic(err)
	}

	eg, _ := errgroup.WithContext(context.Background())

	nextReader, nextWriter := io.Pipe()

	eg.Go(runModule(mInput, nextWriter, nil))

	compression, defined := os.LookupEnv(compressEnv)
	if defined && compression != "" {
		mCompress, err := loadModule(core.ModuleGroupBackup, core.ModuleTypeCompress, compression, compressEnv)
		if err != nil {
			log.Panic(err)
		}
		_nextReader, _nextWriter := io.Pipe()
		eg.Go(runModule(mCompress, _nextWriter, nextReader))
		nextReader = _nextReader
	}

	encryption, defined := os.LookupEnv(encryptEnv)
	if defined && encryption != "" {
		mEncrypt, err := loadModule(core.ModuleGroupBackup, core.ModuleTypeEncrypt, encryption, encryptEnv)
		if err != nil {
			log.Panic(err)
		}
		_nextReader, _nextWriter := io.Pipe()
		eg.Go(runModule(mEncrypt, _nextWriter, nextReader))
		nextReader = _nextReader
	}

	eg.Go(runModule(mOutput, nil, nextReader))

	if err := eg.Wait(); err != nil {
		log.Panic(err)
	}
}

func loadModule(mGroup core.ModuleGroup, mType core.ModuleType, mName, envPrefix string) (core.Module, error) {
	module := core.GetModule(mGroup, mType, mName)
	if module == nil {
		return nil, fmt.Errorf("module %s/%s not found", mType, mName)
	}
	config := module.GetConfig()
	err := envconfig.Process(envPrefix, config)
	if err != nil {
		return nil, fmt.Errorf("module %s/%s env parse err: %s", mType, mName, err)
	}
	log.Printf("module %s/%s config: %+v", mType, mName, config)
	if err := module.InitModule(config); err != nil {
		return nil, fmt.Errorf("init module %s/%s err: %s", mType, mName, err)
	}
	return module, nil
}

func runModule(module core.Module, writer io.WriteCloser, reader io.ReadCloser) func() error {
	return func() error {
		defer func() {
			if writer != nil {
				writer.Close()
			}
			if reader != nil {
				reader.Close()
			}
		}()
		t := time.Now()
		err := module.InitPipe(writer, reader)
		if err != nil {
			return fmt.Errorf("module %s/%s pipe initialization err: %s", module.GetType(), module.GetName(), err)
		}
		err = module.Run()
		if err != nil {
			return fmt.Errorf("module %s/%s execution err: %s", module.GetType(), module.GetName(), err)
		}
		log.Printf("module %s/%s done by %.2fms", module.GetType(), module.GetName(), time.Since(t).Seconds()*1000)
		return nil
	}
}
