package svc

import (
	"testing"

	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/web/pkg/announcement"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

func TestCurrentAnnouncement(t *testing.T) {
	newWeb := func(a *announcement.Store) Web {
		return Web{logger: log.NopLogger(), announcementStore: a}
	}
	newStore := func() *announcement.Store {
		return announcement.NewStore(store.Create(store.Store("memory")))
	}

	t.Run("nil when there is no store", func(t *testing.T) {
		require.Nil(t, newWeb(nil).currentAnnouncement())
	})

	t.Run("nil when the store is empty", func(t *testing.T) {
		require.Nil(t, newWeb(newStore()).currentAnnouncement())
	})

	t.Run("nil when disabled", func(t *testing.T) {
		s := newStore()
		require.NoError(t, s.Set(announcement.Announcement{Enabled: false, BannerText: "hi", InfoText: "info"}))
		require.Nil(t, newWeb(s).currentAnnouncement())
	})

	t.Run("nil when enabled but the banner text is empty", func(t *testing.T) {
		s := newStore()
		require.NoError(t, s.Set(announcement.Announcement{Enabled: true, InfoText: "info"}))
		require.Nil(t, newWeb(s).currentAnnouncement())
	})

	t.Run("returns banner and info text when enabled with a banner text", func(t *testing.T) {
		s := newStore()
		require.NoError(t, s.Set(announcement.Announcement{Enabled: true, BannerText: "hi", InfoText: "info"}))
		require.Equal(t, &config.Announcement{BannerText: "hi", InfoText: "info"}, newWeb(s).currentAnnouncement())
	})
}
