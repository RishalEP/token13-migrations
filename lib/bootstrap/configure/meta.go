package configure

import (
	"context"
	"errors"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/philchia/agollo/v4"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"github.com/yuncai836/boilerplates/otel"
	"go.uber.org/zap"
)

type Apollo struct {
	MetaAddr        string   `yaml:"meta_addr" env:"apollo_addr"`
	AppId           string   `yaml:"app_id" env:"apollo_app_id"`
	Cluster         string   `yaml:"cluster" env:"apollo_cluster"`
	NamespaceNames  []string `yaml:"namespace_names" env:"apollo_namespace"`
	AccesskeySecret string   `yaml:"accesskey_secret" env:"apollo_secret"`
}

func (a *Apollo) IsEmpty() bool {
	return a.MetaAddr == "" ||
		a.AppId == "" ||
		a.Cluster == "" ||
		a.NamespaceNames == nil ||
		a.AccesskeySecret == "" ||
		len(a.NamespaceNames) == 0
}

type ViperConfig struct {
	ConfigPath string `yaml:"path" env:"config_path"`
}

func (v *ViperConfig) IsEmpty() bool {
	return v.ConfigPath == ""
}

func ReadEnvConfig[T any]() *T {
	var a = new(T)
	err := cleanenv.ReadEnv(a)
	if err != nil {
		panic(err)
	}
	return a
}

func Get[T any](namespace string) (*T, error) {
	t, err0 := ApolloGet[T](namespace)
	if err0 == nil {
		return t, nil
	}

	t, err1 := ViperGetAll[T]()
	if err1 == nil {
		return t, nil
	}

	return nil, errors.Join(err0, err1)
}

func MustGet[T any](namespace string) *T {
	if t, err := Get[T](namespace); err != nil {
		panic(err)
	} else {
		return t
	}
}
func ConfigUpdateEventStream[T any](namespace string) (chan *T, func()) {
	var msg = make(chan *T, 1)
	agollo.OnUpdate(func(event *agollo.ChangeEvent) {
		ctx, span := otel.Tracer.Start(context.Background(), "apollo config update")
		defer span.End()
		conf, err := ApolloGet[T](namespace)
		if err != nil {
			otelzap.L().ErrorContext(ctx, "apollo client get config fail", zap.Error(err))
			return
		}
		otelzap.L().InfoContext(ctx, "apollo config update success")
		msg <- conf
	})
	return msg, func() {
		err := agollo.Stop()
		otelzap.L().Warn("stop apollo config sync fail", zap.Error(err))
		close(msg)
	}
}
