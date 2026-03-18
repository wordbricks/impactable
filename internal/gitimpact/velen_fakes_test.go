package gitimpact

import (
	"context"
	"fmt"
)

type fakeVelenClient struct {
	identity   VelenIdentity
	whoAmIErr  error
	currentOrg string
	orgErr     error
	sources    []VelenSource
	listErr    error
	showByKey  map[string]VelenSource
	showErr    error
	queryBody  []byte
	queryErr   error
	queryFunc  func(sourceKey string, queryFile string) ([]byte, error)
}

func (client fakeVelenClient) WhoAmI(context.Context) (VelenIdentity, error) {
	if client.whoAmIErr != nil {
		return VelenIdentity{}, client.whoAmIErr
	}
	return client.identity, nil
}

func (client fakeVelenClient) CurrentOrg(context.Context) (string, error) {
	if client.orgErr != nil {
		return "", client.orgErr
	}
	return client.currentOrg, nil
}

func (client fakeVelenClient) ListSources(context.Context) ([]VelenSource, error) {
	if client.listErr != nil {
		return nil, client.listErr
	}
	return client.sources, nil
}

func (client fakeVelenClient) ShowSource(_ context.Context, sourceKey string) (VelenSource, error) {
	if client.showErr != nil {
		return VelenSource{}, client.showErr
	}
	if source, ok := client.showByKey[sourceKey]; ok {
		return source, nil
	}
	return VelenSource{}, fmt.Errorf("source %q not found", sourceKey)
}

func (client fakeVelenClient) Query(_ context.Context, sourceKey string, queryFile string) ([]byte, error) {
	if client.queryFunc != nil {
		return client.queryFunc(sourceKey, queryFile)
	}
	if client.queryErr != nil {
		return nil, client.queryErr
	}
	return client.queryBody, nil
}

type assertErr string

func (value assertErr) Error() string {
	return string(value)
}
