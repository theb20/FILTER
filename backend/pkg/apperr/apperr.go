// Package apperr porte les erreurs métier entre les services (gRPC) et le gateway (RFC 9457).
//
// Une erreur a un code gRPC (famille) et un code applicatif STABLE du contrat (`VALIDATION_FAILED`,
// `INVALID_CREDENTIALS`…, voir `components.schemas.Problem.code` dans openapi.yaml).
package apperr

import (
	"errors"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Domain identifie la source des ErrorInfo.
const Domain = "filterfinder"

// Codes applicatifs du contrat utilisés par identity et le gateway.
const (
	CodeValidation          = "VALIDATION_FAILED"
	CodeUnauthenticated     = "UNAUTHENTICATED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeRateLimited         = "RATE_LIMITED"
	CodeInvalidCredentials  = "INVALID_CREDENTIALS"
	CodeInvalidCode         = "INVALID_CODE"
	CodePasswordCompromised = "PASSWORD_COMPROMISED"
	CodeEmailNotVerified    = "EMAIL_NOT_VERIFIED"
	CodeCSRFFailed          = "CSRF_FAILED"
	CodeInternal            = "INTERNAL"
	CodeNotImplemented      = "NOT_IMPLEMENTED"
	CodeSelfActionForbidden = "SELF_ACTION_FORBIDDEN"
	CodeLastSuperAdmin      = "LAST_SUPER_ADMIN"
)

// Field décrit une erreur de champ (pointeur JSON).
type Field struct {
	Pointer string
	Code    string
	Message string
}

// Error est une erreur métier typée.
type Error struct {
	Kind       codes.Code
	Code       string
	Message    string
	RetryAfter time.Duration
	Fields     []Field
	cause      error
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }
func (e *Error) Unwrap() error { return e.cause }

// New construit une erreur.
func New(kind codes.Code, code, msg string) *Error {
	return &Error{Kind: kind, Code: code, Message: msg}
}

// WithCause conserve la cause interne (jamais exposée au client).
func (e *Error) WithCause(err error) *Error { e.cause = err; return e }

// WithRetryAfter indique le délai avant une nouvelle tentative (429).
func (e *Error) WithRetryAfter(d time.Duration) *Error { e.RetryAfter = d; return e }

// WithFields ajoute des erreurs de champ.
func (e *Error) WithFields(f ...Field) *Error { e.Fields = append(e.Fields, f...); return e }

func Validation(msg string, fields ...Field) *Error {
	return New(codes.InvalidArgument, CodeValidation, msg).WithFields(fields...)
}
func Unauthenticated(code, msg string) *Error    { return New(codes.Unauthenticated, code, msg) }
func NotFound(msg string) *Error                 { return New(codes.NotFound, CodeNotFound, msg) }
func PermissionDenied(code, msg string) *Error   { return New(codes.PermissionDenied, code, msg) }
func FailedPrecondition(code, msg string) *Error { return New(codes.FailedPrecondition, code, msg) }
func Internal(cause error) *Error {
	return New(codes.Internal, CodeInternal, "erreur interne").WithCause(cause)
}
func RateLimited(retry time.Duration) *Error {
	return New(codes.ResourceExhausted, CodeRateLimited, "trop de requêtes").WithRetryAfter(retry)
}

// ToGRPC convertit une erreur en statut gRPC avec ErrorInfo (`reason` = code applicatif), RetryInfo et BadRequest.
// Une erreur inconnue devient INTERNAL sans exposer son détail.
func ToGRPC(err error) error {
	if err == nil {
		return nil
	}
	var ae *Error
	if !errors.As(err, &ae) {
		if _, ok := status.FromError(err); ok {
			return err
		}
		ae = Internal(err)
	}
	st := status.New(ae.Kind, ae.Message)
	withDetails, derr := st.WithDetails(&errdetails.ErrorInfo{Reason: ae.Code, Domain: Domain})
	if derr == nil {
		st = withDetails
	}
	if ae.RetryAfter > 0 {
		if s, e := st.WithDetails(&errdetails.RetryInfo{RetryDelay: durationpb.New(ae.RetryAfter)}); e == nil {
			st = s
		}
	}
	if len(ae.Fields) > 0 {
		br := &errdetails.BadRequest{}
		for _, f := range ae.Fields {
			br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{Field: f.Pointer, Description: f.Message, Reason: f.Code})
		}
		if s, e := st.WithDetails(br); e == nil {
			st = s
		}
	}
	return st.Err()
}

// FromGRPC reconstruit une Error depuis une erreur gRPC (côté gateway). Les erreurs non gRPC deviennent INTERNAL.
func FromGRPC(err error) *Error {
	if err == nil {
		return nil
	}
	var ae *Error
	if errors.As(err, &ae) {
		return ae
	}
	st, ok := status.FromError(err)
	if !ok {
		return Internal(err)
	}
	out := &Error{Kind: st.Code(), Message: st.Message(), Code: defaultCode(st.Code())}
	for _, d := range st.Details() {
		switch v := d.(type) {
		case *errdetails.ErrorInfo:
			if v.GetReason() != "" {
				out.Code = v.GetReason()
			}
		case *errdetails.RetryInfo:
			out.RetryAfter = v.GetRetryDelay().AsDuration()
		case *errdetails.BadRequest:
			for _, fv := range v.GetFieldViolations() {
				out.Fields = append(out.Fields, Field{Pointer: fv.GetField(), Code: fv.GetReason(), Message: fv.GetDescription()})
			}
		}
	}
	return out
}

func defaultCode(c codes.Code) string {
	switch c {
	case codes.InvalidArgument:
		return CodeValidation
	case codes.Unauthenticated:
		return CodeUnauthenticated
	case codes.PermissionDenied:
		return CodeForbidden
	case codes.NotFound:
		return CodeNotFound
	case codes.AlreadyExists, codes.Aborted:
		return CodeConflict
	case codes.ResourceExhausted:
		return CodeRateLimited
	case codes.Unimplemented:
		return CodeNotImplemented
	default:
		return CodeInternal
	}
}
