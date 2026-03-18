package gitimpact

import "sync"

var velenClientFactoryMu sync.Mutex

func withVelenClientFactory(factory func() VelenClient, run func()) {
	velenClientFactoryMu.Lock()
	previous := newVelenClient
	newVelenClient = factory
	defer func() {
		newVelenClient = previous
		velenClientFactoryMu.Unlock()
	}()
	run()
}
