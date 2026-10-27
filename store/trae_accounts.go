package store

import (
	"errors"
	"fmt"
)

var ErrTraeAccountNotFound = errors.New("账号不存在")

func (s *Store) ListTraeAccounts() ([]TraeAccount, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]TraeAccount{}, s.config.TraeAccounts...), s.config.TraeActiveAccount
}

// ChangeTraeAccount is one disk transaction. Session refreshes update the active
// profile through SetTraeSession, so a later switch never restores an old token.
func (s *Store) ChangeTraeAccount(action, id, name string, session TraeSession) (TraeSession, error) {
	creating := action == "save" && id == ""
	if action == "save" && id == "" {
		var err error
		id, err = GenerateID()
		if err != nil {
			return TraeSession{}, err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old := *s.config
	s.config.TraeAccounts = append([]TraeAccount{}, old.TraeAccounts...)
	index := -1
	for i, a := range s.config.TraeAccounts {
		if a.ID == id {
			index = i
			break
		}
	}
	var err error
	switch action {
	case "save":
		if !creating && index < 0 {
			err = ErrTraeAccountNotFound
			break
		}
		entry := TraeAccount{ID: id, Name: name, Session: session}
		if index < 0 {
			s.config.TraeAccounts = append(s.config.TraeAccounts, entry)
		} else {
			s.config.TraeAccounts[index] = entry
		}
		s.config.TraeActiveAccount = id
		s.config.TraeSession = &session
		s.config.TraeToken = ""
		s.config.TraeUserID = ""
	case "activate":
		if index < 0 {
			err = ErrTraeAccountNotFound
			break
		}
		session = s.config.TraeAccounts[index].Session
		if session.Token == "" {
			err = fmt.Errorf("此账号没有可用令牌，请先更新凭据")
			break
		}
		s.config.TraeActiveAccount = id
		s.config.TraeSession = &session
		s.config.TraeToken = ""
		s.config.TraeUserID = ""
	case "delete":
		if index < 0 {
			err = ErrTraeAccountNotFound
			break
		}
		s.config.TraeAccounts = append(s.config.TraeAccounts[:index], s.config.TraeAccounts[index+1:]...)
		if s.config.TraeActiveAccount == id {
			s.config.TraeActiveAccount = ""
			session = TraeSession{}
			s.config.TraeSession = &session
			s.config.TraeToken = ""
			s.config.TraeUserID = ""
		}
	default:
		err = fmt.Errorf("不支持的账号操作")
	}
	if err == nil {
		err = s.saveLocked()
	}
	if err != nil {
		*s.config = old
		return TraeSession{}, err
	}
	if s.config.TraeSession != nil {
		return *s.config.TraeSession, nil
	}
	return TraeSession{Token: s.config.TraeToken, UserID: s.config.TraeUserID}, nil
}
