package model

type Option struct {
	OptionID    int64  `db:"option_id"`
	OptionName  string `db:"option_name"`
	OptionValue string `db:"option_value"`
	Autoload    string `db:"autoload"`
}
