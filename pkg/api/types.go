package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v2"
)

type BigBang struct {
	Packages      []BigBangPackage
	Configuration Configuration
}

func (BigBang) ElementType() reflect.Type {
	return reflect.TypeOf((*BigBang)(nil)).Elem()
}

type BigBangPackage interface {

	//Get list of network policies needed for each namespace
	NetworkPolicies() []string //need to update to network policies needed to interact with servicemesh

	Enabled() bool

	Deploy(*pulumi.Context, BigBang, ...pulumi.Resource) ([]pulumi.Resource, error)

	GetResources() ([]pulumi.Resource, error)

	GetViolations() *Violations

	// Get the namespace the package is deployed into.  This could be multiple
	// e.g. in the case of Istio
	// GetNamespace() string

	// Pass in another package and have this package capture how
	// the namespace(s) for that package should be adjusted.
	// Think additional NetworkPolicies, AuthorizationPolicies
	// etc
	// MutateNamepsace(BigBangPackage) ([]pulumi.Resource, error)
}

type Ingress struct {
	ServiceName pulumi.Output
	Port        pulumi.Output
	//defaults to the subdomain of the domain
	Name string
	// Namespace
	Namespace string
	//overrides the default of ${Name}.domain
	Hostname string
	// gateway to attach to
	Gateway string
}

type Violations struct {
	// Which Docker registries does this chart need access to?
	AllowedDockerRegistries []string
	// Which pods violioate policy for not mounting the host file system?
	AllowedHostFilesystem []string
	//which pods need access to mount the Host Networking?
	NoHostNamespace []string

	// Which Daemonsets/pods are allowed to have tolderations for restricted taints
	RestrictedTaint []string

	SELinuxPolicy []string

	VolumeTypes []string

	RunAsRoot []string
}

type Logging int

const (
	LoggingPLG Logging = iota
	LoggingELK
	LoggingNone
)

type Policy string

const (
	PolicyGatekeeper Policy = "gatekeeper"
	PolicyKyverno    Policy = "kyverno"
	PolicyNone       Policy = "none"
)

// func (s Policy) String() string {
// 	return toString[s]
// }

// var toString = map[Policy]string{
// 	PolicyGatekeeper: "gatekeeper",
// 	PolicyKyverno:    "kyverno",
// 	PolicyNone:       "none",
// }

// var toID = map[string]Policy{
// 	"gatekeeper": PolicyGatekeeper,
// 	"kyverno":    PolicyKyverno,
// 	"none":       PolicyNone,
// }

// // MarshalJSON marshals the enum as a quoted json string
// func (s Policy) MarshalJSON() ([]byte, error) {
// 	buffer := bytes.NewBufferString(`"`)
// 	buffer.WriteString(toString[s])
// 	buffer.WriteString(`"`)
// 	return buffer.Bytes(), nil
// }

// // UnmarshalJSON unmashals a quoted json string to the enum value
// func (s *Policy) UnmarshalJSON(b []byte) error {
// 	var j string
// 	err := json.Unmarshal(b, &j)
// 	if err != nil {
// 		return err
// 	}
// 	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
// 	*s = toID[j]
// 	return nil
// }

type ServiceMesh string

const (
	ServieMeshIstio    ServiceMesh = "istio"
	ServiceMeshLinkerd             = "linkerd"
	ServiceMeshNone                = "none"
)

type Monitoring string

const (
	MonitoringPrometheus Monitoring = "prometheus"
	MonitoringNewRelic   Monitoring = "newrelic"
	MonitoringNone       Monitoring = "none"
)

type RuntimeSecurity int

const (
	RuntimeSecurityNeuvector RuntimeSecurity = iota
	RuntimeSecurityPrismaCloud
	RuntimeSecurityNone
)

type PolicyConfiguration struct {
	Name       Policy `yaml:"name"`
	Enforce    bool   `yaml:"enforce"`
	Violations struct {
		ApprovedRegistries struct {
			Enabled             bool
			NamespaceExceptions []string
		}
		HostNetwork struct {
			Enabled             bool
			NamespaceExceptions []string
			ResourceExceptions  []string
		}
	} `yaml:"violations,omitempty"`
}

type ServiceMeshConfiguration struct {
	Name     ServiceMesh `yaml:"name"` //istio or linkderd
	Domain   string      //where to host things
	Gateways []struct {
		Name string
		Tls  struct {
			Key      string `yaml:"key"`
			KeyFile  string `yaml:"keyFile"`
			Cert     string `yaml:"cert"`
			CertFile string `yaml:"certFile"`
		} `yaml:"tls"`
		Domain string //default to be the ServiceMesh.Domain
	} `yaml:"gateways"`

	CommonConfig CommonConfig
}

type LoggingConfiguration struct {
	Name Logging `yaml:"name"`

	CommonConfig CommonConfig
}

type MonitoringConfiguration struct {
	Name Monitoring `yaml:"name"`

	CommonConfig CommonConfig
}

type RuntimeSecurityConfiguration struct {
	Name         RuntimeSecurity `yaml:"name"`
	CommonConfig CommonConfig
}

type CommonConfig struct {
	ImagePullSecrets []struct {
		Registry string `yaml:"registry"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"imagePullSecrets"`
	NetworkPolicies struct {
		Enabled          bool   `yaml:"enabled"`
		ControlPlaneCIDR string `yaml:"controlPlaneCIDR"`
	} `yaml:"networkPolicies"`
	Namespace string `yaml:"namespace,omitempty"` //Defaults to package name, e.g. bigbang for BigBang
}

type Configuration struct {
	Policy          PolicyConfiguration          `yaml:"policy,omitempty"`
	ServiceMesh     ServiceMeshConfiguration     `yaml:"serviceMesh,omitempty"`
	Logging         LoggingConfiguration         `yaml:"logging,omitempty"`
	Monitoring      MonitoringConfiguration      `yaml:"monitoring,omitempty"`
	RuntimeSecurity RuntimeSecurityConfiguration `yaml:"runtimeSecurity,omitempty"`
	CommonConfig    `yaml:"global,omitempty"`
	Development     bool `yaml:"development,omitempty"`
}

func (c Configuration) ToString() string {
	b, _ := json.Marshal(c)
	return string(b)
}

func NewConfiguration(s string) Configuration {
	c := Configuration{}
	json.Unmarshal([]byte(s), &c)
	return c
}

// func ()

func (Configuration) ElementType() reflect.Type {
	return reflect.TypeOf((*Configuration)(nil)).Elem()
}

func LoadConfiguration(filename string) (*Configuration, error) {
	c := Configuration{}
	yamlFile, err := ioutil.ReadFile(filename)
	fmt.Println(string(yamlFile))
	if err != nil {
		return &c, err
	}
	err = yaml.Unmarshal(yamlFile, &c)
	return &c, err
}
