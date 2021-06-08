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
 * Create: 2021-05-29
 * Description: eggo command configs implement
 ******************************************************************************/

package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v1"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/constants"
	"gitee.com/openeuler/eggo/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	masterPackages = []*api.Packages{
		{
			Name: "kubernetes-client",
			Type: "repo",
		},
		{
			Name: "kubernetes-master",
			Type: "repo",
		},
		{
			Name: "kubernetes-kubeadm",
			Type: "repo",
		},
		{
			Name: "coredns",
			Type: "repo",
		},
		{
			Name: "tar",
			Type: "repo",
		},
	}

	masterExports = []*api.OpenPorts{
		// kube-apiserver
		{
			Port:     6443,
			Protocol: "tcp",
		},
		// kube-scheduler
		{
			Port:     10251,
			Protocol: "tcp",
		},
		// kube-controller-manager
		{
			Port:     10252,
			Protocol: "tcp",
		},
		// coredns
		{
			Port:     53,
			Protocol: "tcp",
		},
		// coredns
		{
			Port:     53,
			Protocol: "udp",
		},
		// coredns
		{
			Port:     9153,
			Protocol: "tcp",
		},
	}

	nodePackages = []*api.Packages{
		{
			Name: "docker-engine",
			Type: "repo",
		},
		{
			Name: "iSulad",
			Type: "repo",
		},
		{
			Name: "kubernetes-client",
			Type: "repo",
		},
		{
			Name: "kubernetes-node",
			Type: "repo",
		},
		{
			Name: "kubernetes-kubelet",
			Type: "repo",
		},
		{
			Name: "tar",
			Type: "repo",
		},
	}

	nodeExports = []*api.OpenPorts{
		// kubelet
		{
			Port:     10250,
			Protocol: "tcp",
		},
		// kube-proxy
		{
			Port:     10256,
			Protocol: "tcp",
		},
	}

	etcdPackages = []*api.Packages{
		{
			Name: "etcd",
			Type: "repo",
		},
		{
			Name: "tar",
			Type: "repo",
		},
	}

	etcdExports = []*api.OpenPorts{
		// etcd api
		{
			Port:     2379,
			Protocol: "tcp",
		},
		// etcd peer
		{
			Port:     2380,
			Protocol: "tcp",
		},
		// etcd metric
		{
			Port:     2381,
			Protocol: "tcp",
		},
	}

	loadbalancePackages = []*api.Packages{
		{
			Name: "nginx",
			Type: "repo",
		},
		{
			Name: "tar",
			Type: "repo",
		},
	}
)

type ConfigExtraArgs struct {
	Name      string            `yaml:"name"`
	ExtraArgs map[string]string `yaml:"extra-args"`
}

type Package struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`    // repo, pkg, binary
	Dst  string `yaml:"dstpath"` // used only when type is binary
}

type HostConfig struct {
	Name string `yaml:"name"`
	Ip   string `yaml:"ip"`
	Port int    `yaml:"port"`
	Arch string `yaml:"arch"` // amd64, aarch64, default amd64
}

type deployConfig struct {
	ClusterID         string                   `yaml:"cluster-id"`
	Username          string                   `yaml:"username"`
	Password          string                   `yaml:"password"`
	Masters           []*HostConfig            `yaml:"masters"`
	Nodes             []*HostConfig            `yaml:"nodes"`
	Etcds             []*HostConfig            `yaml:"etcds"`
	LoadBalances      []*HostConfig            `yaml:"loadbalances"`
	ConfigDir         string                   `yaml:"config-dir"`
	CertificateDir    string                   `yaml:"certificate-dir"`
	ExternalCA        bool                     `yaml:"external-ca"`
	ExternalCAPath    string                   `yaml:"external-ca-path"`
	Service           api.ServiceClusterConfig `yaml:"service"`
	NetWork           api.NetworkConfig        `yaml:"network"`
	ApiServerEndpoint string                   `yaml:"apiserver-endpoint"`
	ApiServerCertSans api.Sans                 `yaml:"apiserver-cert-sans"`
	ApiServerTimeout  string                   `yaml:"apiserver-timeout"`
	EtcdExternal      bool                     `yaml:"etcd-external"`
	EtcdToken         string                   `yaml:"etcd-token"`
	EtcdDataDir       string                   `yaml:"etcd-data-dir"`
	DnsVip            string                   `yaml:"dns-vip"`
	DnsDomain         string                   `yaml:"dns-domain"`
	PauseImage        string                   `yaml:"pause-image"`
	NetworkPlugin     string                   `yaml:"network-plugin"`
	CniBinDir         string                   `yaml:"cni-bin-dir"`
	Runtime           string                   `yaml:"runtime"`
	RuntimeEndpoint   string                   `yaml:"runtime-endpoint"`
	ConfigExtraArgs   []*ConfigExtraArgs       `yaml:"config-extra-args"`
	Addons            []*api.AddonConfig       `yaml:"addons"`
	PackageSrc        api.PackageSrcConfig     `yaml:"package-src"`
	Packages          map[string][]*Package    `yaml:"pacakges"` // key: master, node, etcd, loadbalance
}

func getEggoDir() string {
	return filepath.Join(utils.GetSysHome(), ".eggo")
}

func init() {
	if _, err := os.Stat(getEggoDir()); err == nil {
		return
	}

	if err := os.Mkdir(getEggoDir(), 0700); err != nil {
		logrus.Errorf("mkdir eggo directory %v failed", getEggoDir())
	}
}

func getDefaultDeployConfig() string {
	return filepath.Join(getEggoDir(), "deploy.yaml")
}

func loadDeployConfig(file string) (*deployConfig, error) {
	yamlStr, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	conf := &deployConfig{}
	if err := yaml.Unmarshal([]byte(yamlStr), conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func getDefaultClusterdeploymentConfig() *api.ClusterConfig {
	return &api.ClusterConfig{
		Name:      "k8s-cluster",
		ConfigDir: constants.DefaultK8SRootDir,
		Certificate: api.CertificateConfig{
			SavePath: constants.DefaultK8SCertDir,
		},
		ServiceCluster: api.ServiceClusterConfig{
			CIDR:    "10.32.0.0/16",
			DNSAddr: "10.32.0.10",
			Gateway: "10.32.0.1",
		},
		Network: api.NetworkConfig{
			PodCIDR:    "10.244.64.0/16",
			PluginArgs: make(map[string]string),
		},
		LocalEndpoint: api.APIEndpoint{
			AdvertiseAddress: "127.0.0.1",
			BindPort:         6443,
		},
		ControlPlane: api.ControlPlaneConfig{
			ApiConf: &api.ApiServer{
				Timeout: "120s",
			},
			KubeletConf: &api.Kubelet{
				DnsVip:        "10.32.0.10",
				DnsDomain:     "cluster.local",
				PauseImage:    "k8s.gcr.io/pause:3.2",
				NetworkPlugin: "cni",
				CniBinDir:     "/usr/libexec/cni",
			},
		},
		PackageSrc: &api.PackageSrcConfig{
			Type:   "tar.gz",
			ArmSrc: "./pacakges-arm.tar.gz",
			X86Src: "./packages-x86.tar.gz",
		},
		EtcdCluster: api.EtcdClusterConfig{
			Token:    "etcd-cluster",
			DataDir:  "/var/lib/etcd/default.etcd",
			CertsDir: constants.DefaultK8SCertDir,
			External: false,
		},
		DeployDriver: "binary",
	}
}

func createCommonHostConfig(userHostconfig *HostConfig, defaultName string, username string,
	password string) *api.HostConfig {
	arch, name, port := "amd64", defaultName, 22
	if userHostconfig.Arch != "" {
		arch = userHostconfig.Arch
	}
	if userHostconfig.Name != "" {
		name = userHostconfig.Name
	}
	if userHostconfig.Port != 0 {
		port = userHostconfig.Port
	}

	hostconfig := &api.HostConfig{
		Arch:     arch,
		Name:     name,
		Address:  userHostconfig.Ip,
		Port:     port,
		UserName: username,
		Password: password,
	}

	return hostconfig
}

func portExist(openPorts []*api.OpenPorts, port *api.OpenPorts) bool {
	for _, p := range openPorts {
		if p.Protocol == port.Protocol && p.Port == port.Port {
			return true
		}
	}
	return false
}

func addPackagesAndExports(hostconfig *api.HostConfig, pkgs []*api.Packages,
	openPorts []*api.OpenPorts) {
	for _, port := range openPorts {
		if portExist(hostconfig.OpenPorts, port) {
			continue
		}
		hostconfig.OpenPorts = append(hostconfig.OpenPorts, port)
	}

	if hostconfig.Packages == nil {
		hostconfig.Packages = []*api.Packages{}
	}

	noDupPkgs := make(map[string]bool, len(hostconfig.Packages))
	for _, p := range hostconfig.Packages {
		noDupPkgs[p.Name] = true
	}

	for _, pkg := range pkgs {
		if _, ok := noDupPkgs[pkg.Name]; ok {
			continue
		}
		hostconfig.Packages = append(hostconfig.Packages, pkg)
	}
}

func addUserPackages(hostconfig *api.HostConfig, userPkgs []*Package) {
	if hostconfig.Packages == nil {
		hostconfig.Packages = []*api.Packages{}
	}

	noDupPkgs := make(map[string]int, len(hostconfig.Packages))
	for i, p := range hostconfig.Packages {
		noDupPkgs[p.Name] = i
	}

	for _, pkg := range userPkgs {
		p := &api.Packages{
			Name: pkg.Name,
			Type: pkg.Type,
			Dst:  pkg.Dst,
		}
		if i, ok := noDupPkgs[pkg.Name]; ok {
			hostconfig.Packages[i] = p
			continue
		}
		hostconfig.Packages = append(hostconfig.Packages, p)
	}
}

func getPortFromEndPoint(endpoint string) int {
	defaultPort := 6443

	// endpoint:
	// https://192.168.0.11:6443
	uri, err := url.Parse(endpoint)
	if err != nil {
		return defaultPort
	}

	// host:
	// 192.168.0.11:6443
	items := strings.Split(uri.Host, ":")
	if len(items) < 2 {
		return defaultPort
	}

	port, err := strconv.Atoi(items[len(items)-1])
	if err != nil {
		return defaultPort
	}

	return port
}

func fillHostConfig(ccfg *api.ClusterConfig, conf *deployConfig) {
	var hostconfig *api.HostConfig
	var exist bool
	cache := make(map[string]*api.HostConfig)

	for i, master := range conf.Masters {
		hostconfig = createCommonHostConfig(master, "k8s-master-"+strconv.Itoa(i), conf.Username, conf.Password)
		hostconfig.Type |= api.Master
		addUserPackages(hostconfig, conf.Packages["master"])
		addPackagesAndExports(hostconfig, masterPackages, masterExports)
		cache[hostconfig.Address] = hostconfig
	}

	for i, node := range conf.Nodes {
		hostconfig, exist = cache[node.Ip]
		if !exist {
			hostconfig = createCommonHostConfig(node, "k8s-node-"+strconv.Itoa(i), conf.Username,
				conf.Password)
		}
		hostconfig.Type |= api.Worker
		addUserPackages(hostconfig, conf.Packages["node"])
		addPackagesAndExports(hostconfig, nodePackages, nodeExports)
		cache[hostconfig.Address] = hostconfig
	}

	// if no etcd configed, default to install to master
	var etcds []*HostConfig
	if len(conf.Etcds) == 0 {
		etcds = conf.Masters
	} else {
		etcds = conf.Etcds
	}
	for i, etcd := range etcds {
		hostconfig, exist = cache[etcd.Ip]
		if !exist {
			hostconfig = createCommonHostConfig(etcd, "etcd-"+strconv.Itoa(i), conf.Username, conf.Password)
		}
		hostconfig.Type |= api.ETCD
		addUserPackages(hostconfig, conf.Packages["etcd"])
		addPackagesAndExports(hostconfig, etcdPackages, etcdExports)
		cache[hostconfig.Address] = hostconfig
	}

	for i, lb := range conf.LoadBalances {
		hostconfig, exist = cache[lb.Ip]
		if !exist {
			hostconfig = createCommonHostConfig(lb, "k8s-loadbalance-"+strconv.Itoa(i), conf.Username,
				conf.Password)
		}
		hostconfig.Type |= api.LoadBalance

		addUserPackages(hostconfig, conf.Packages["loadbalance"])
		addPackagesAndExports(hostconfig, loadbalancePackages, []*api.OpenPorts{
			{
				Port:     getPortFromEndPoint(conf.ApiServerEndpoint),
				Protocol: "tcp",
			},
		})
		cache[hostconfig.Address] = hostconfig
	}

	for _, v := range cache {
		ccfg.Nodes = append(ccfg.Nodes, v)
	}
}

func setIfStrConfigNotEmpty(config *string, userConfig string) {
	if config == nil {
		logrus.Errorf("invalid nil config")
		return
	}
	if userConfig != "" {
		*config = userConfig
	}
}

func setStrStrMap(config map[string]string, userConfig map[string]string) {
	if config == nil {
		logrus.Errorf("invalid nil config")
		return
	}
	for k, v := range userConfig {
		config[k] = v
	}
}

func notInStrArray(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return false
		}
	}
	return true
}

func setStrArray(config []string, userConfig []string) {
	for _, v := range userConfig {
		if notInStrArray(config, v) {
			config = append(config, v)
		}
	}
}

func toClusterdeploymentConfig(conf *deployConfig) *api.ClusterConfig {
	ccfg := getDefaultClusterdeploymentConfig()

	setIfStrConfigNotEmpty(&ccfg.Name, conf.ClusterID)
	fillHostConfig(ccfg, conf)
	setIfStrConfigNotEmpty(&ccfg.ConfigDir, conf.ConfigDir)
	setIfStrConfigNotEmpty(&ccfg.Certificate.SavePath, conf.CertificateDir)
	ccfg.Certificate.ExternalCA = conf.ExternalCA
	setIfStrConfigNotEmpty(&ccfg.Certificate.ExternalCAPath, conf.ExternalCAPath)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.CIDR, conf.Service.CIDR)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.DNSAddr, conf.Service.DNSAddr)
	setIfStrConfigNotEmpty(&ccfg.ServiceCluster.Gateway, conf.Service.Gateway)
	setIfStrConfigNotEmpty(&ccfg.Network.PodCIDR, conf.NetWork.PodCIDR)
	setIfStrConfigNotEmpty(&ccfg.Network.Plugin, conf.NetWork.Plugin)
	setStrStrMap(ccfg.Network.PluginArgs, conf.NetWork.PluginArgs)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.Endpoint, conf.ApiServerEndpoint)
	setStrArray(ccfg.ControlPlane.ApiConf.CertSans.DNSNames, conf.ApiServerCertSans.DNSNames)
	setStrArray(ccfg.ControlPlane.ApiConf.CertSans.IPs, conf.ApiServerCertSans.IPs)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.ApiConf.Timeout, conf.ApiServerTimeout)
	ccfg.EtcdCluster.External = conf.EtcdExternal
	for _, node := range ccfg.Nodes {
		if (node.Type & api.ETCD) != 0 {
			ccfg.EtcdCluster.Nodes = append(ccfg.EtcdCluster.Nodes, node)
		}
	}
	setIfStrConfigNotEmpty(&ccfg.EtcdCluster.Token, conf.EtcdToken)
	setIfStrConfigNotEmpty(&ccfg.EtcdCluster.DataDir, conf.EtcdDataDir)
	setIfStrConfigNotEmpty(&ccfg.PackageSrc.Type, conf.PackageSrc.Type)
	setIfStrConfigNotEmpty(&ccfg.PackageSrc.ArmSrc, conf.PackageSrc.ArmSrc)
	setIfStrConfigNotEmpty(&ccfg.PackageSrc.X86Src, conf.PackageSrc.X86Src)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.KubeletConf.DnsVip, conf.DnsVip)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.KubeletConf.DnsDomain, conf.DnsDomain)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.KubeletConf.PauseImage, conf.PauseImage)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.KubeletConf.NetworkPlugin, conf.NetworkPlugin)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.KubeletConf.CniBinDir, conf.CniBinDir)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.KubeletConf.Runtime, conf.Runtime)
	setIfStrConfigNotEmpty(&ccfg.ControlPlane.KubeletConf.RuntimeEndpoint, conf.RuntimeEndpoint)

	ccfg.Addons = append(ccfg.Addons, conf.Addons...)

	return ccfg
}

func getHostconfigs(format string, ips []string) []*HostConfig {
	var confs []*HostConfig
	for i, ip := range ips {
		confs = append(confs, &HostConfig{
			Name: fmt.Sprintf(format, i),
			Ip:   ip,
			Port: 22,
			Arch: "amd64",
		})
	}
	return confs
}

func createDeployConfigTemplate(file string) error {
	var masters, nodes, etcds, lbs []*HostConfig
	masterIP := []string{"192.168.0.2"}
	if opts.masters != nil {
		masterIP = opts.masters
	}
	nodesIP := []string{"192.168.0.3", "192.168.0.4"}
	if opts.nodes != nil {
		nodesIP = opts.nodes
	}
	lbsIP := []string{"192.168.0.1"}
	if opts.loadbalancer != nil {
		lbsIP = opts.loadbalancer
	}
	etcdsIP := masterIP
	if opts.etcds != nil {
		etcdsIP = opts.etcds
	}
	masters = getHostconfigs("k8s-master-%d", masterIP)
	nodes = getHostconfigs("k8s-node-%d", nodesIP)
	etcds = getHostconfigs("etcd-%d", etcdsIP)
	lbs = getHostconfigs("k8s-loadbalance-%d", lbsIP)
	if etcds == nil {
		etcds = masters
	}

	conf := &deployConfig{
		ClusterID:      opts.name,
		Username:       opts.username,
		Password:       opts.password,
		Masters:        masters,
		Nodes:          nodes,
		Etcds:          etcds,
		LoadBalances:   lbs,
		ConfigDir:      constants.DefaultK8SRootDir,
		CertificateDir: constants.DefaultK8SCertDir,
		ExternalCA:     false,
		ExternalCAPath: "/opt/externalca",
		Service: api.ServiceClusterConfig{
			CIDR:    "10.32.0.0/16",
			DNSAddr: "10.32.0.10",
			Gateway: "10.32.0.1",
		},
		NetWork: api.NetworkConfig{
			PodCIDR:    "10.244.64.0/16",
			PluginArgs: make(map[string]string),
		},
		ApiServerEndpoint: fmt.Sprintf("https://%s:6443", lbs[0].Ip),
		ApiServerCertSans: api.Sans{},
		ApiServerTimeout:  "120s",
		EtcdExternal:      false,
		EtcdToken:         "etcd-cluster",
		EtcdDataDir:       "/var/lib/etcd/default.etcd",
		DnsVip:            "10.32.0.10",
		DnsDomain:         "cluster.local",
		PauseImage:        "k8s.gcr.io/pause:3.2",
		NetworkPlugin:     "cni",
		CniBinDir:         "/usr/libexec/cni",
		Runtime:           "iSulad",
		RuntimeEndpoint:   "unix:///var/run/isulad.sock",
		Addons: []*api.AddonConfig{
			{
				Type:     "file",
				Filename: "calico.yaml",
			},
		},
		PackageSrc: api.PackageSrcConfig{
			Type:   "tar.gz",
			ArmSrc: "/root/pkgs/pacakges-arm.tar.gz",
			X86Src: "/root/pkgs/packages-x86.tar.gz",
		},
		Packages: map[string][]*Package{
			"master": {
				&Package{
					Name: "kubernetes-client",
					Type: "pkg",
				},
				&Package{
					Name: "kubernetes-master",
					Type: "pkg",
				},
				&Package{
					Name: "kubernetes-kubeadm",
					Type: "pkg",
				},
				&Package{
					Name: "coredns",
					Type: "pkg",
				},
				&Package{
					Name: "tar",
					Type: "repo",
				},
				&Package{
					Name: "addons",
					Type: "binary",
					Dst:  "/etc/kubernetes",
				},
			},
			"node": {
				&Package{
					Name: "conntrack-tools",
					Type: "pkg",
				},
				&Package{
					Name: "socat",
					Type: "pkg",
				},
				&Package{
					Name: "containernetworking-plugins",
					Type: "pkg",
				},
				&Package{
					Name: "tar",
					Type: "repo",
				},
				&Package{
					Name: "emacs-filesystem",
					Type: "pkg",
				},
				&Package{
					Name: "gflags",
					Type: "pkg",
				},
				&Package{
					Name: "gpm-libs",
					Type: "pkg",
				},
				&Package{
					Name: "http-parser",
					Type: "pkg",
				},
				&Package{
					Name: "libwebsockets",
					Type: "pkg",
				},
				&Package{
					Name: "re2",
					Type: "pkg",
				},
				&Package{
					Name: "rsync",
					Type: "pkg",
				},
				&Package{
					Name: "vim-filesystem",
					Type: "pkg",
				},
				&Package{
					Name: "vim-common",
					Type: "pkg",
				},
				&Package{
					Name: "vim-enhanced",
					Type: "pkg",
				},
				&Package{
					Name: "yajl",
					Type: "pkg",
				},
				&Package{
					Name: "zlib-devel",
					Type: "pkg",
				},
				&Package{
					Name: "protobuf",
					Type: "pkg",
				},
				&Package{
					Name: "protobuf-devel",
					Type: "pkg",
				},
				&Package{
					Name: "grpc",
					Type: "pkg",
				},
				&Package{
					Name: "lxc",
					Type: "pkg",
				},
				&Package{
					Name: "lxc-libs",
					Type: "pkg",
				},
				&Package{
					Name: "lcr",
					Type: "pkg",
				},
				&Package{
					Name: "clibcni",
					Type: "pkg",
				},
				&Package{
					Name: "libcgroup",
					Type: "pkg",
				},
				&Package{
					Name: "docker-engine",
					Type: "pkg",
				},
				&Package{
					Name: "iSulad",
					Type: "pkg",
				},
				&Package{
					Name: "kubernetes-client",
					Type: "pkg",
				},
				&Package{
					Name: "kubernetes-node",
					Type: "pkg",
				},
				&Package{
					Name: "kubernetes-kubelet",
					Type: "pkg",
				},
			},
			"etcd": {
				&Package{
					Name: "etcd",
					Type: "pkg",
				},
				&Package{
					Name: "tar",
					Type: "repo",
				},
			},
			"loadbalance": {
				&Package{
					Name: "nginx",
					Type: "pkg",
				},
				&Package{
					Name: "gd",
					Type: "pkg",
				},
				&Package{
					Name: "gperftools-libs",
					Type: "pkg",
				},
				&Package{
					Name: "libunwind",
					Type: "pkg",
				},
				&Package{
					Name: "libwebp",
					Type: "pkg",
				},
				&Package{
					Name: "libxslt",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-all-modules",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-filesystem",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-http-image-filter",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-http-perl",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-http-xslt-filter",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-mail",
					Type: "pkg",
				},
				&Package{
					Name: "nginx-mod-stream",
					Type: "pkg",
				},
				&Package{
					Name: "tar",
					Type: "repo",
				},
			},
		},
	}

	d, err := yaml.Marshal(conf)
	if err != nil {
		return fmt.Errorf("marshal template config failed: %v", err)
	}

	if err := ioutil.WriteFile(file, d, 0700); err != nil {
		return fmt.Errorf("write template config file failed: %v", err)
	}

	return nil
}
