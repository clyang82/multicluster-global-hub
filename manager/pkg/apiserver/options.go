package apiserver

import (
	"github.com/spf13/pflag"
	genericapiserveroptions "k8s.io/apiserver/pkg/server/options"
)

type Options struct {
	KubeConfigFile string
	ServerRun      *genericapiserveroptions.ServerRunOptions
	SecureServing  *genericapiserveroptions.SecureServingOptionsWithLoopback
	Authentication *genericapiserveroptions.DelegatingAuthenticationOptions
	Authorization  *genericapiserveroptions.DelegatingAuthorizationOptions
}

func NewOptions() *Options {
	return &Options{
		KubeConfigFile: "",
		ServerRun:      genericapiserveroptions.NewServerRunOptions(),
		SecureServing:  genericapiserveroptions.NewSecureServingOptions().WithLoopback(),
		Authentication: genericapiserveroptions.NewDelegatingAuthenticationOptions(),
		Authorization:  genericapiserveroptions.NewDelegatingAuthorizationOptions(),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.KubeConfigFile, "kube-config-file", o.KubeConfigFile, "Kubernetes configuration file to connect to kube-apiserver")
	o.ServerRun.AddUniversalFlags(fs)
	o.SecureServing.AddFlags(fs)
	o.Authentication.AddFlags(fs)
	o.Authorization.AddFlags(fs)
}
