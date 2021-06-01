/******************************************************************************
 * Copyright (c) Huawei Technologies Co., Ltd. 2021. All rights reserved.
 * eggo licensed under the Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *     http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
 * PURPOSE.
 * See the Mulan PSL v2 for more details.
 * Author: wangfengtu
 * Create: 2021-05-31
 * Description: cmd configs testcase
 ******************************************************************************/

package main

import (
	"fmt"
	"gopkg.in/yaml.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCmdConfigs(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "cmd-configs-test-")
	if err != nil {
		t.Fatalf("create tempdir for cmd configs failed: %v", err)
	}
	defer os.RemoveAll(tempdir)

	f := filepath.Join(tempdir, "config.yaml")

	if err := createDeployConfigTemplate(f); err != nil {
		t.Fatalf("create deploy template config file failed: %v", err)
	}

	conf, err := loadDeployConfig(f)
	if err != nil {
		t.Fatalf("load deploy config file failed: %v", err)
	}

	ccfg := toClusterdeploymentConfig(conf)
	d, err := yaml.Marshal(ccfg)
	if err != nil {
		t.Fatalf("marshal cluster config failed: %v", err)
	}

	fmt.Printf("%v\n", string(d))
}