package commontools

import (
	"fmt"
	"os"
	"path/filepath"

	"gitee.com/openeuler/eggo/pkg/api"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	"github.com/sirupsen/logrus"
)

var (
	ETCDRequiredCerts = []string{
		"kube-apiserver-etcd-client.crt",
		"kube-apiserver-etcd-client.key",
		"etcd/ca.key",
		"etcd/ca.crt",
	}
	MasterRequiredCerts = []string{
		"sa.pub",
		"sa.key",
		"ca.crt",
		"ca.key",
		"front-proxy-ca.crt",
		"front-proxy-ca.key",
	}
	WokerRequiredCerts = []string{
		"ca.crt",
	}
)

type CopyCaCertificatesTask struct {
	Cluster *api.ClusterConfig
}

func (ct *CopyCaCertificatesTask) Name() string {
	return "CopyCaCertificatesTask"
}

func checkCaExists(cluster string, requireCerts []string) bool {
	for _, cert := range requireCerts {
		_, err := os.Lstat(filepath.Join(api.GetCertificateStorePath(cluster), cert))
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func getRequireCerts(hostType uint16) []string {
	tmpCerts := make(map[string]struct{}, 1)
	if (hostType & api.Master) != 0 {
		for _, cert := range MasterRequiredCerts {
			tmpCerts[cert] = struct{}{}
		}
	}
	if (hostType & api.Worker) != 0 {
		for _, cert := range WokerRequiredCerts {
			tmpCerts[cert] = struct{}{}
		}
	}
	if (hostType & api.ETCD) != 0 {
		for _, cert := range ETCDRequiredCerts {
			tmpCerts[cert] = struct{}{}
		}
	}
	var ret []string
	for k, _ := range tmpCerts {
		ret = append(ret, k)
	}
	return ret
}

func (ct *CopyCaCertificatesTask) Run(r runner.Runner, hcf *api.HostConfig) error {
	requireCerts := getRequireCerts(hcf.Type)
	if !checkCaExists(ct.Cluster.Name, requireCerts) {
		return fmt.Errorf("[certs] cannot find ca certificates")
	}
	cmd := fmt.Sprintf("sudo -E /bin/sh -c \"mkdir -p %s\"", ct.Cluster.Certificate.SavePath)
	if (hcf.Type & api.ETCD) != 0 {
		cmd = fmt.Sprintf("sudo -E /bin/sh -c \"mkdir -p %s/etcd\"", ct.Cluster.Certificate.SavePath)
	}

	if _, err := r.RunCommand(cmd); err != nil {
		return err
	}

	homeDir := api.GetCertificateStorePath(ct.Cluster.Name)
	for _, cert := range requireCerts {
		if err := r.Copy(filepath.Join(homeDir, cert), filepath.Join(ct.Cluster.Certificate.SavePath, cert)); err != nil {
			logrus.Errorf("copy cert: %s to host: %s failed: %v", cert, hcf.Name, err)
			return err
		}
	}
	logrus.Infof("copy certs to host: %s success", hcf.Name)

	return nil
}
