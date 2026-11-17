// Package grpcserver expose le service content en gRPC (contrat proto/contentadmin/v1).
package grpcserver

import (
	"context"

	"github.com/theb20/FILTER/backend/pkg/apperr"
	commonv1 "github.com/theb20/FILTER/backend/pkg/gen/common/v1"
	pb "github.com/theb20/FILTER/backend/pkg/gen/contentadmin/v1"
	"github.com/theb20/FILTER/backend/services/content/internal/application"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Server struct {
	pb.UnimplementedContentAdminServiceServer
	Svc *application.Service
}

func meta(ctx context.Context, key string) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if v := md.Get(key); len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func actor(ctx context.Context) application.Actor {
	a := application.Actor{IP: meta(ctx, "x-client-ip"), TraceID: meta(ctx, "x-request-id")}
	if meta(ctx, "x-actor-type") == "staff" { // un visiteur n'est jamais « acteur » : l'audit le note comme système
		a.ID = meta(ctx, "x-actor-id")
	}
	return a
}

func toDoc(d *application.Document) *pb.Document {
	s, _ := structpb.NewStruct(d.Data)
	return &pb.Document{Id: d.ID, Kind: d.Kind, Data: s, Version: int32(d.Version), CreatedAt: timestamppb.New(d.CreatedAt), UpdatedAt: timestamppb.New(d.UpdatedAt)}
}

func (s *Server) ListDocuments(ctx context.Context, r *pb.ListDocumentsRequest) (*pb.ListDocumentsResponse, error) {
	p, err := s.Svc.List(ctx, r.GetKind(), r.GetQuery(), r.GetWhere(), int(r.GetPage().GetPage()), int(r.GetPage().GetPageSize()))
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	out := &pb.ListDocumentsResponse{Page: &commonv1.PageResponse{Page: int32(p.Page), PageSize: int32(p.PageSize), Total: int64(p.Total), TotalPages: int32(p.Pages)}}
	for _, d := range p.Items {
		out.Items = append(out.Items, toDoc(d))
	}
	return out, nil
}

func (s *Server) GetDocument(ctx context.Context, r *pb.GetDocumentRequest) (*pb.GetDocumentResponse, error) {
	d, err := s.Svc.Get(ctx, r.GetKind(), r.GetIdOrKey())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.GetDocumentResponse{Document: toDoc(d)}, nil
}

func (s *Server) CreateDocument(ctx context.Context, r *pb.CreateDocumentRequest) (*pb.CreateDocumentResponse, error) {
	d, err := s.Svc.Create(ctx, actor(ctx), r.GetKind(), r.GetData().AsMap())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.CreateDocumentResponse{Document: toDoc(d)}, nil
}

func (s *Server) UpdateDocument(ctx context.Context, r *pb.UpdateDocumentRequest) (*pb.UpdateDocumentResponse, error) {
	d, err := s.Svc.Update(ctx, actor(ctx), r.GetKind(), r.GetId(), r.GetData().AsMap(), r.GetReplace())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.UpdateDocumentResponse{Document: toDoc(d)}, nil
}

func (s *Server) DeleteDocument(ctx context.Context, r *pb.DeleteDocumentRequest) (*pb.DeleteDocumentResponse, error) {
	if err := s.Svc.Delete(ctx, actor(ctx), r.GetKind(), r.GetId()); err != nil {
		return nil, apperr.ToGRPC(err)
	}
	return &pb.DeleteDocumentResponse{}, nil
}

func (s *Server) GetSettings(ctx context.Context, r *pb.GetSettingsRequest) (*pb.GetSettingsResponse, error) {
	st, err := s.Svc.GetSettings(ctx, r.GetKey())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	d, _ := structpb.NewStruct(st.Data)
	return &pb.GetSettingsResponse{Data: d, Version: int32(st.Version), UpdatedAt: timestamppb.New(st.UpdatedAt)}, nil
}

func (s *Server) PutSettings(ctx context.Context, r *pb.PutSettingsRequest) (*pb.PutSettingsResponse, error) {
	st, err := s.Svc.PutSettings(ctx, actor(ctx), r.GetKey(), r.GetData().AsMap())
	if err != nil {
		return nil, apperr.ToGRPC(err)
	}
	d, _ := structpb.NewStruct(st.Data)
	return &pb.PutSettingsResponse{Data: d, Version: int32(st.Version), UpdatedAt: timestamppb.New(st.UpdatedAt)}, nil
}
