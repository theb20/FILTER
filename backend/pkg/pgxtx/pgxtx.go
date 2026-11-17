// Package pgxtx fournit l'unité de travail transactionnelle partagée par tous les services : les dépôts lisent la
// transaction dans le contexte, ce qui permet d'écrire la donnée métier ET l'événement (outbox) dans la MÊME transaction.
package pgxtx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX est satisfait par *pgxpool.Pool et pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ctxKey struct{}

// With place une transaction dans le contexte.
func With(ctx context.Context, db DBTX) context.Context { return context.WithValue(ctx, ctxKey{}, db) }

// From renvoie la transaction du contexte, ou nil.
func From(ctx context.Context) DBTX {
	db, _ := ctx.Value(ctxKey{}).(DBTX)
	return db
}

// Querier renvoie la transaction du contexte si elle existe, sinon le pool.
func Querier(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if db := From(ctx); db != nil {
		return db
	}
	return pool
}

// Run exécute fn dans une transaction : validation si fn réussit, annulation sinon (erreur ou panique). Si le contexte
// porte déjà une transaction, fn la REJOINT (pas de transaction imbriquée).
func Run(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context) error) (err error) {
	if From(ctx) != nil {
		return fn(ctx)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(r)
		}
		if err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()
	if err = fn(With(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
