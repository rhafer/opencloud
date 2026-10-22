package parser

import (
	"errors"
	"fmt"
	"strings"

	occfg "github.com/opencloud-eu/opencloud/pkg/config"
	"github.com/opencloud-eu/opencloud/pkg/kql"
	"github.com/opencloud-eu/opencloud/pkg/shared"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config"
	"github.com/opencloud-eu/opencloud/services/search/pkg/config/defaults"

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

func Validate(cfg *config.Config) error {
	if cfg.Events.Disabled && cfg.GRPC.Disabled {
		return shared.AllComponentsDisabledError(cfg.Service.Name)
	}

	if cfg.TokenManager.JWTSecret == "" {
		return shared.MissingJWTTokenError(cfg.Service.Name)
	}

	if cfg.ServiceAccount.ServiceAccountID == "" {
		return shared.MissingServiceAccountID(cfg.Service.Name)
	}
	if cfg.ServiceAccount.ServiceAccountSecret == "" {
		return shared.MissingServiceAccountSecret(cfg.Service.Name)
	}

	return validateQueryFilter(cfg.QueryFilter)
}

func validateQueryFilter(query string) error {
	if query == "" {
		return nil
	}

	expanded := strings.NewReplacer(
		config.PersonalSpaceRootPlaceholder, `"`+config.NoResourceID+`"`,
		config.ProjectSpaceRootPlaceholder, `"`+config.NoResourceID+`"`,
	).Replace(query)

	if strings.Contains(expanded, "{{") || strings.Contains(expanded, "}}") {
		return errors.New("search query filter: unknown placeholder")
	}

	if _, err := (kql.Builder{}).Build(expanded); err != nil {
		return fmt.Errorf("search query filter: %w", err)
	}

	return nil
}
