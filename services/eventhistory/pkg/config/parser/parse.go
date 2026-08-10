package parser

import (
	"errors"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config"
	"github.com/opencloud-eu/opencloud/services/eventhistory/pkg/config/defaults"

	"github.com/opencloud-eu/opencloud/pkg/config/envdecode"
)

// ParseConfig loads configuration from known paths.
func ParseConfig(cfg *config.Config) error {
	err := occfg.BindSourcesToStructs(cfg.Service.Name, cfg)
	if err != nil {
		return err
	}

	defaults.EnsureDefaults(cfg)

	// load all env variables relevant to the config in the current context.
	if err := envdecode.Decode(cfg); err != nil {
		// no environment variable set for this config is an expected "error"
		if !errors.Is(err, envdecode.ErrNoTargetFieldsAreSet) {
			return err
		}
	}

	defaults.Sanitize(cfg)

	return Validate(cfg)
}

// Validate validates the config
func Validate(cfg *config.Config) error {
	// this need to be discussed, I put it here for now because I belive we need to ensure that the service is doing
	// at least something, if we have events and grpc disabled that means that the service is doing actually nothing
	// which can be a siletlly ignorred and cause issues in the future
	if cfg.Events.Disabled && cfg.GRPC.Disabled {
		return errors.New("both events and gRPC are disabled; at least one must be enabled")
	}

	return nil
}
