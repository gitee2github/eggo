package commontools

import (
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
	"time"

	"gitee.com/openeuler/eggo/pkg/clusterdeployment"
	"gitee.com/openeuler/eggo/pkg/utils/runner"
	kkutil "github.com/kubesphere/kubekey/pkg/util"
	"github.com/lithammer/dedent"
	"github.com/sirupsen/logrus"
	bootstraputil "k8s.io/cluster-bootstrap/token/util"
)

const (
	TokenTemplate = `apiVersion: v1
kind: Secret
metadata:
	name: bootstrap-token-{{ .ID }}
	namespace: kube-system
type: bootstrap.kubernetes.io/token
stringData:
	description: "{{ .Description }}"
	token-id: {{ .ID }}
	token-secret: {{ .Secret }}
	expiration: {{ .Expiration }}
	{{- range $i, $v := .Usages }}
	$v
	{{- end }}
	auth-extra-groups: {{ .AuthExtraGroups }}
`
)

func CreateBootstrapToken(r runner.Runner, bconf *clusterdeployment.BootstrapTokenConfig) error {
	var sb strings.Builder
	var usages []string
	now := time.Now()
	tmpl := template.Must(template.New("bootstrap token").Parse(dedent.Dedent(TokenTemplate)))
	datastore := map[string]interface{}{}
	datastore["Description"] = bconf.Description
	datastore["ID"] = bconf.ID
	datastore["Secret"] = bconf.Secret
	ttl := *bconf.TTL
	if bconf.TTL == nil {
		// default set ttl 24 hours
		ttl = 24 * time.Hour
	}
	datastore["Expiration"] = now.Add(ttl).Format(time.RFC3339)
	for _, usage := range bconf.Usages {
		usages = append(usages, fmt.Sprintf("usage-bootstrap-%s: true", usage))
	}
	datastore["Usages"] = usages
	if len(bconf.AuthExtraGroups) > 0 {
		datastore["AuthExtraGroups"] = strings.Join(bconf.AuthExtraGroups, ",")
	}
	coreConfig, err := kkutil.Render(tmpl, datastore)
	if err != nil {
		logrus.Errorf("rend core config failed: %v", err)
		return err
	}
	sb.WriteString("sudo -E /bin/sh -c \"mkdir /tmp/.eggo")
	tokenYamlBase64 := base64.StdEncoding.EncodeToString([]byte(coreConfig))
	sb.WriteString(fmt.Sprintf(" && echo %s | base64 -d > /tmp/.eggo/bootstrap_token.yaml", tokenYamlBase64))
	sb.WriteString(" && kubectl apply -f /tmp/.eggo/bootstrap_token.yaml")
	sb.WriteString("\"")

	_, err = r.RunCommand(sb.String())
	if err != nil {
		logrus.Errorf("create core config failed: %v", err)
		return err
	}
	return nil
}

func CreateBootstrapTokensForCluster(r runner.Runner, ccfg *clusterdeployment.ClusterConfig) error {
	for _, token := range ccfg.BootStrapTokens {
		if err := CreateBootstrapToken(r, token); err != nil {
			logrus.Errorf("create bootstrap token failed: %v", err)
			return err
		}
	}
	return nil
}

func GenerateBootstrapToken() (token, id, secret string, err error) {
	tokenStr, err := bootstraputil.GenerateBootstrapToken()
	if err != nil {
		logrus.Errorf("generate bootstrap token string error: %v", err)
		return "", "", "", err
	}
	splitStrs := bootstraputil.BootstrapTokenRegexp.FindStringSubmatch(tokenStr)
	if len(splitStrs) != 3 {
		logrus.Errorf("generate bootstrap token string invalid: %s", tokenStr)
		return "", "", "", fmt.Errorf("generate bootstrap token string invalid: %s", tokenStr)
	}

	return splitStrs[0], splitStrs[1], splitStrs[2], nil
}