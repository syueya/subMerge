package systemsettings

import "github.com/submerge/submerge/backend/internal/outbound"

type ProxyAdapter struct{ manager *Manager }

func NewProxyAdapter(manager *Manager) *ProxyAdapter { return &ProxyAdapter{manager: manager} }
func (a *ProxyAdapter) View() outbound.View          { return a.manager.proxyView() }
func (a *ProxyAdapter) Save(req outbound.UpdateRequest) (outbound.View, error) {
	return a.manager.SaveProxy(req)
}
func (a *ProxyAdapter) Reset() (outbound.View, error) { return a.manager.ResetProxy() }
