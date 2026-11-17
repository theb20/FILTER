package server

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseTime(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }

func isConflict(err error) bool { return status.Code(err) == codes.AlreadyExists }
func isNotFound(err error) bool { return status.Code(err) == codes.NotFound }
