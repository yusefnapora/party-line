package components

import (
	"fmt"
	"github.com/maxence-charriere/go-app/v7/pkg/app"
	"github.com/yusefnapora/party-line/types"
)

type PeerListView struct {
	app.Compo

	users []*types.UserInfo

	newPeerRequested func(string)
}

func PeerList(users []*types.UserInfo, onNewPeerRequested func(string)) *PeerListView {
	return &PeerListView{
		users: users,
		newPeerRequested: onNewPeerRequested,
	}
}

func (v *PeerListView) Render() app.UI {
	return app.Div().Class("peer-list-view").Body(
		app.H3().Body(app.Text("Peers")),

		app.Range(v.users).Slice(func(i int) app.UI {
			return UserCard(v.users[i])
		}),

		app.Input().Class("new-peer-input").
			Placeholder("Enter a peer id / multiaddr to connect").OnChange(v.newPeerTextChanged),
		)
}

func (v *PeerListView) newPeerTextChanged(ctx app.Context, e app.Event) {
	text := ctx.JSSrc.Get("value").String()
	if v.newPeerRequested != nil {
		v.newPeerRequested(text)
	}
	ctx.JSSrc.Set("value", "")
}

func (v *PeerListView) SetUsers(users []*types.UserInfo) {
	v.users = users
	v.Update()
}

func (v *PeerListView) AddUser(info *types.UserInfo) {
	for _, i := range v.users {
		if i.PeerId == info.PeerId {
			return
		}
	}

	v.users = append(v.users, info)
	v.Update()
}

type UserAvatarView struct {
	app.Compo

	user *types.UserInfo
	size int
}

func UserAvatar(info *types.UserInfo, size int) *UserAvatarView {
	return &UserAvatarView{
		user: info,
		size: size,
	}
}

func (v *UserAvatarView) Render() app.UI {
	// go-app doesn't have Svg tags, so we construct raw html
	const cssClass = "user-avatar"
	html := fmt.Sprintf(`<svg data-jdenticon-value="%s" width="%d" height="%d" class="%s">Avatar for %s</svg>`,
		v.user.PeerId, v.size, v.size, cssClass, v.user.Nickname)

	return app.Raw(html)
}

type UserCardView struct {
	app.Compo

	user *types.UserInfo
}

func UserCard(user *types.UserInfo) *UserCardView {
	return &UserCardView{user: user}
}

func (v *UserCardView) Render() app.UI {
	idlen := len(v.user.PeerId)
	shortID := v.user.PeerId[idlen-8 : idlen]

	const aviSize = 64

	return app.Div().Class("user-card").Body(
		UserAvatar(v.user, aviSize),

		app.Div().Class("user-card-text").Body(
			app.Span().Class("user-card-nickname").Body(
				app.Text(v.user.Nickname)),

			app.Span().Class("user-card-peerid").Body(
				app.Text(shortID)),
		),
	)
}
