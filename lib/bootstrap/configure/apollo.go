package configure

import (
	"bootstrap/telemetry"
	"github.com/philchia/agollo/v4"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"strings"
)

var l = telemetry.NewLogger()

func InitApolloClient(logger agollo.Logger) error {
	var a = ReadEnvConfig[Apollo]()
	appConfig := agollo.Conf{
		AppID:           a.AppId,
		Cluster:         a.Cluster,
		NameSpaceNames:  a.NamespaceNames,
		MetaAddr:        a.MetaAddr,
		AccesskeySecret: a.AccesskeySecret,
	}
	l.Info("Apollo Config:", zap.String("Appid:", a.AppId), zap.String("Cluster:", a.Cluster), zap.String("metaAddress:", a.MetaAddr), zap.String("AccesskeySecret:", a.AccesskeySecret))
	for _, namespace := range a.NamespaceNames {
		l.Info("Apollo Namespaces:", zap.String("namespace:", namespace))
	}
	return agollo.Start(&appConfig, agollo.WithLogger(logger), agollo.SkipLocalCache())
}

func ApolloGet[T any](namespace string) (*T, error) {
	var content = agollo.GetContent(agollo.WithNamespace(namespace))
	var t = new(T)
	de := yaml.NewDecoder(strings.NewReader(content))
	de.KnownFields(true)
	var err = de.Decode(t)
	return t, err
}

func ApolloMustGet[T any](namespace string) *T {
	t, err := ApolloGet[T](namespace)
	if err != nil {
		panic(err)
	}
	return t
}
