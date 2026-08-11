package identity

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	ldapv3 "github.com/go-ldap/ldap/v3"
	ocldap "github.com/opencloud-eu/opencloud/pkg/ldap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/registry"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/reva/v2/pkg/rgrpc/todo/pool"
	"github.com/opencloud-eu/reva/v2/pkg/utils/ldap"
	"go.opentelemetry.io/otel/trace"
)

func CreateIdentityBackends(name string, cfg *config.Config, logger *log.Logger, traceProvider trace.TracerProvider) (Backend, EducationBackend, error) {
	switch name {
	case "cs3":
		gatewaySelector, err := pool.GatewaySelector(
			cfg.Reva.Address,
			append(
				cfg.Reva.GetRevaOptions(),
				pool.WithRegistry(registry.GetRegistry()),
				pool.WithTracerProvider(traceProvider),
			)...,
		)
		if err != nil {
			return nil, nil, err
		}

		return &CS3{
			Config:          cfg.Reva,
			Logger:          logger,
			GatewaySelector: gatewaySelector,
		}, nil, nil
	case "ldap":
		var err error

		var tlsConf *tls.Config
		if cfg.Identity.LDAP.Insecure {
			// When insecure is set to true then we don't need a certificate.
			cfg.Identity.LDAP.CACert = ""
			tlsConf = &tls.Config{
				MinVersion: tls.VersionTLS12,

				//nolint:gosec // We need the ability to run with "insecure" (dev/testing)
				InsecureSkipVerify: cfg.Identity.LDAP.Insecure,
			}
		}

		if cfg.Identity.LDAP.CACert != "" {
			if err := ocldap.WaitForCA(*logger,
				cfg.Identity.LDAP.Insecure,
				cfg.Identity.LDAP.CACert); err != nil {
				logger.Fatal().Err(err).Msg("The configured LDAP CA cert does not exist")
			}
			if tlsConf == nil {
				tlsConf = &tls.Config{
					MinVersion: tls.VersionTLS12,
				}
			}
			certs := x509.NewCertPool()
			pemData, err := os.ReadFile(cfg.Identity.LDAP.CACert)
			if err != nil {
				logger.Error().Err(err).Msg("Error initializing LDAP Backend")
				return nil, nil, err
			}
			if !certs.AppendCertsFromPEM(pemData) {
				logger.Error().Msg("Error initializing LDAP Backend. Adding CA cert failed")
				return nil, nil, err
			}
			tlsConf.RootCAs = certs
		}

		conn := ldap.NewLDAPWithReconnect(
			ldap.Config{
				URI:          cfg.Identity.LDAP.URI,
				BindDN:       cfg.Identity.LDAP.BindDN,
				BindPassword: cfg.Identity.LDAP.BindPassword,
				TLSConfig:    tlsConf,
			},
		)
		conn.SetLogger(&logger.Logger)
		lb, err := NewLDAPBackend(conn, cfg.Identity.LDAP, logger)
		if err != nil {
			logger.Error().Err(err).Msg("Error initializing LDAP Backend")
			return nil, nil, err
		}

		identityBackend := lb
		var eduBackend EducationBackend = lb

		if !cfg.Identity.LDAP.EducationResourcesEnabled {
			eduBackend = &ErrEducationBackend{}
		}

		disableMechanismType, err := ParseDisableMechanismType(cfg.Identity.LDAP.DisableUserMechanism)
		if err != nil {
			logger.Error().Err(err).Msg("Error initializing LDAP Backend")
			return nil, nil, err
		}

		if disableMechanismType == DisableMechanismGroup {
			logger.Info().Msg("LocalUserDisable is true, will create group if not exists")
			err := lb.CreateLDAPGroupByDN(cfg.Identity.LDAP.LdapDisabledUsersGroupDN)
			if err != nil {
				isAnError := false
				var lerr *ldapv3.Error
				if errors.As(err, &lerr) {
					if lerr.ResultCode != ldapv3.LDAPResultEntryAlreadyExists {
						isAnError = true
					}
				} else {
					isAnError = true
				}

				if isAnError {
					msg := "error adding group for disabling users"
					logger.Error().Err(err).Str("local_user_disable", cfg.Identity.LDAP.LdapDisabledUsersGroupDN).Msg(msg)
					return nil, nil, err
				}
			}
		}

		return identityBackend, eduBackend, nil

	default:
		err := fmt.Errorf("unknown identity backend: '%s'", name)
		logger.Err(err)
		return nil, nil, err
	}
}
