package configure

import (
	"github.com/philchia/agollo/v4"
	"gopkg.in/yaml.v3"
	"strings"
)

func MustInitApolloClient(logger agollo.Logger) {
	err := InitApolloClient(logger)
	if err != nil {
		panic(err)
	}
}

func InitApolloClient(logger agollo.Logger) error {
	var a = ReadEnvConfig[Apollo]()
	appConfig := agollo.Conf{
		AppID:           a.AppId,
		Cluster:         a.Cluster,
		NameSpaceNames:  a.NamespaceNames,
		MetaAddr:        a.MetaAddr,
		AccesskeySecret: a.AccesskeySecret,
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
