package repository

import (
	"context"
	"database/sql"
	"fmt"

	"press/internal/errors"
	"press/internal/model"

	"github.com/jmoiron/sqlx"
)

type OptionsRepository struct {
	db *sqlx.DB
}

func NewOptionsRepository(db *sqlx.DB) *OptionsRepository {
	return &OptionsRepository{db: db}
}

func (r *OptionsRepository) Get(ctx context.Context, name string) (*model.Option, error) {
	option := &model.Option{}
	err := r.db.GetContext(ctx, option, "SELECT * FROM wp_options WHERE option_name = ?", name)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound(errors.ErrOptionNotFound, fmt.Sprintf("Option not found: %s", name))
	}
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return option, nil
}

func (r *OptionsRepository) GetValue(ctx context.Context, name string) (string, error) {
	opt, err := r.Get(ctx, name)
	if err != nil {
		return "", err
	}
	return opt.OptionValue, nil
}

// GetValueDefault returns the option value, or defaultValue if the option
// doesn't exist. Only returns an error for actual database failures.
func (r *OptionsRepository) GetValueDefault(ctx context.Context, name, defaultValue string) (string, error) {
	opt, err := r.Get(ctx, name)
	if errors.IsCode(err, errors.ErrOptionNotFound) {
		return defaultValue, nil
	}
	if err != nil {
		return "", err
	}
	return opt.OptionValue, nil
}

func (r *OptionsRepository) Set(ctx context.Context, name, value string, autoload string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO wp_options (option_name, option_value, autoload)
		VALUES (?, ?, ?)
		ON CONFLICT(option_name) DO UPDATE SET option_value = excluded.option_value, autoload = excluded.autoload`,
		name, value, autoload,
	)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}

func (r *OptionsRepository) Delete(ctx context.Context, name string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM wp_options WHERE option_name = ?", name)
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.NotFound(errors.ErrOptionNotFound, fmt.Sprintf("Option not found: %s", name))
	}
	return nil
}

func (r *OptionsRepository) List(ctx context.Context) ([]model.Option, error) {
	var options []model.Option
	err := r.db.SelectContext(ctx, &options, "SELECT * FROM wp_options ORDER BY option_name")
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	return options, nil
}

func (r *OptionsRepository) GetAutoloadOptions(ctx context.Context) (map[string]string, error) {
	var options []model.Option
	err := r.db.SelectContext(ctx, &options, "SELECT * FROM wp_options WHERE autoload = 'yes'")
	if err != nil {
		return nil, errors.Internal(err, errors.ErrQueryFailed)
	}
	result := make(map[string]string, len(options))
	for _, opt := range options {
		result[opt.OptionName] = opt.OptionValue
	}
	return result, nil
}

func (r *OptionsRepository) SetMultiple(ctx context.Context, options map[string]string, autoload string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	defer tx.Rollback()

	for name, value := range options {
		_, err := tx.Exec(`
			INSERT INTO wp_options (option_name, option_value, autoload)
			VALUES (?, ?, ?)
			ON CONFLICT(option_name) DO UPDATE SET option_value = excluded.option_value, autoload = excluded.autoload`,
			name, value, autoload,
		)
		if err != nil {
			return errors.Internal(err, errors.ErrQueryFailed)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Internal(err, errors.ErrQueryFailed)
	}
	return nil
}
