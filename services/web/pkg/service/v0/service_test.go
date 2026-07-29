package svc

import (
	"testing"

	"github.com/opencloud-eu/reva/v2/pkg/store"
	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/web/pkg/announcement"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

func TestManagedAnnouncement(t *testing.T) {
	newWeb := func(a *announcement.Store) Web {
		return Web{logger: log.NopLogger(), announcementStore: a}
	}
	newStore := func() *announcement.Store {
		return announcement.NewStore(store.Create(store.Store("memory")))
	}

	t.Run("not managed when there is no store, so a static config is kept", func(t *testing.T) {
		a, managed := newWeb(nil).managedAnnouncement()
		require.Nil(t, a)
		require.False(t, managed)
	})

	t.Run("not managed when the store is empty, so a static config is kept", func(t *testing.T) {
		a, managed := newWeb(newStore()).managedAnnouncement()
		require.Nil(t, a)
		require.False(t, managed)
	})

	t.Run("managed but nil when disabled, so a static config is cleared", func(t *testing.T) {
		s := newStore()
		require.NoError(t, s.Set(announcement.Announcement{Enabled: false, BannerText: "hi", InfoText: "info"}))
		a, managed := newWeb(s).managedAnnouncement()
		require.Nil(t, a)
		require.True(t, managed)
	})

	t.Run("not managed when enabled but the banner text is empty", func(t *testing.T) {
		s := newStore()
		require.NoError(t, s.Set(announcement.Announcement{Enabled: true, InfoText: "info"}))
		a, managed := newWeb(s).managedAnnouncement()
		require.Nil(t, a)
		require.False(t, managed)
	})

	t.Run("managed with banner and info text when enabled with a banner text", func(t *testing.T) {
		s := newStore()
		require.NoError(t, s.Set(announcement.Announcement{Enabled: true, BannerText: "hi", InfoText: "info"}))
		a, managed := newWeb(s).managedAnnouncement()
		require.Equal(t, &config.Announcement{BannerText: "hi", InfoText: "info"}, a)
		require.True(t, managed)
	})
}
