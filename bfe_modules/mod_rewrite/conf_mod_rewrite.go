// Copyright (c) 2019 Baidu, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mod_rewrite

import (
	gcfg "gopkg.in/gcfg.v1"
	"github.com/baidu/go-lib/log"
)

import (
	"github.com/baidu/bfe/bfe_util"
)

type ConfModReWrite struct {
	Basic struct {
		DataPath string // path of config data (rewrite)
	}

	Log struct {
		OpenDebug bool
	}
}

// ConfLoad loades config from config file
func ConfLoad(filePath string, confRoot string) (*ConfModReWrite, error) {
	var err error
	var cfg ConfModReWrite

	// read config from file
	err = gcfg.ReadFileInto(&cfg, filePath)
	if err != nil {
		return &cfg, err
	}

	// check conf of mod_rewrite
	err = cfg.Check(confRoot)
	if err != nil {
		return &cfg, err
	}

	return &cfg, nil
}

func (cfg *ConfModReWrite) Check(confRoot string) error {
	return ConfModReWriteCheck(cfg, confRoot)
}

func ConfModReWriteCheck(cfg *ConfModReWrite, confRoot string) error {
	if cfg.Basic.DataPath == "" {
		log.Logger.Warn("ModReWrite.DataPath not set, use default value")
		cfg.Basic.DataPath = "mod_rewrite/rewrite.data"
	}

	cfg.Basic.DataPath = bfe_util.ConfPathProc(cfg.Basic.DataPath, confRoot)
	return nil
}
