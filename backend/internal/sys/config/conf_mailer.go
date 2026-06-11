package config

import "encoding/json"

type MailerConf struct {
	Host     string `json:"host"     conf:""`
	Port     int    `json:"port"     conf:""`
	Username string `json:"username" conf:""`
	Password string `json:"password" conf:""`
	From     string `json:"from"     conf:""`
}

func (m MailerConf) MarshalJSON() ([]byte, error) {
	type alias MailerConf
	a := alias(m)
	if a.Password != "" {
		a.Password = redactedValue
	}
	return json.Marshal(a)
}

// Ready is a simple check to ensure that the configuration is not empty.
// or with it's default state.
func (mc *MailerConf) Ready() bool {
	return mc.Host != "" && mc.Port != 0 && mc.Username != "" && mc.Password != "" && mc.From != ""
}
