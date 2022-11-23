package tandem

import (
	"github.com/SSSOC-CAN/laniakea/intercept"
	"github.com/TheRebelOfBabylon/tandem/config"
	"github.com/rs/zerolog"
)

func Main(cfg *config.Config, log zerolog.Logger, interceptor *intercept.Interceptor) error {
	// We need to start the WebSocket server
	// Listen for incoming connections
	// Manage ongoing connections
	// Run the Nostr protocol overtop the WebSocket connections
	// Listen for key Nostr events
	// Take appropriate actions
	<-interceptor.ShutdownChannel()
	return nil
}
