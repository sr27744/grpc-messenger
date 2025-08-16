package main

import (
  "context"
  "log"
  "net"
  "sync"
  "time"

  chatv1 "github.com/sr27744/grpc-messenger/server/gen/chat/v1"
  "google.golang.org/grpc"
)

type room struct {
  mu   sync.Mutex
  subs map[chan *chatv1.StreamEnvelope]struct{}
  hist []*chatv1.ChatMessage
}
type hub struct {
  mu    sync.Mutex
  rooms map[string]*room
}
func newHub() *hub { return &hub{rooms: map[string]*room{}} }
func (h *hub) get(id string) *room {
  h.mu.Lock(); defer h.mu.Unlock()
  r, ok := h.rooms[id]
  if !ok { r = &room{subs: map[chan *chatv1.StreamEnvelope]struct{}{}}
    h.rooms[id] = r }
  return r
}

type chatServer struct {
  chatv1.UnimplementedChatServiceServer
  h *hub
}

func now() int64 { return time.Now().Unix() }

func (s *chatServer) Send(ctx context.Context, req *chatv1.SendRequest) (*chatv1.SendResponse, error) {
  r := s.h.get(req.GetRoomId())
  msg := &chatv1.ChatMessage{
    Id:         time.Now().Format("20060102150405.000"),
    RoomId:     req.GetRoomId(),
    Sender:     &chatv1.User{Id: "browser", DisplayName: "browser"},
    Text:       req.GetText(),
    SentAtUnix: now(),
  }
  r.mu.Lock()
  r.hist = append(r.hist, msg)
  if len(r.hist) > 200 { r.hist = r.hist[1:] }
  for ch := range r.subs {
    select { case ch <- &chatv1.StreamEnvelope{Payload: &chatv1.StreamEnvelope_Message{Message: msg}}:
    default: }
  }
  r.mu.Unlock()
  return &chatv1.SendResponse{Message: msg}, nil
}

func (s *chatServer) History(ctx context.Context, req *chatv1.HistoryRequest) (*chatv1.HistoryResponse, error) {
  r := s.h.get(req.GetRoomId())
  r.mu.Lock(); defer r.mu.Unlock()
  hist := r.hist
  if n := int(req.GetLimit()); n > 0 && n < len(hist) { hist = hist[len(hist)-n:] }
  return &chatv1.HistoryResponse{Messages: hist}, nil
}

func (s *chatServer) ChatStream(req *chatv1.HistoryRequest, stream chatv1.ChatService_ChatStreamServer) error {
  r := s.h.get(req.GetRoomId())
  ch := make(chan *chatv1.StreamEnvelope, 64)

  r.mu.Lock()
  r.subs[ch] = struct{}{}
  if req.GetLimit() > 0 {
    n := int(req.GetLimit()); if n > len(r.hist) { n = len(r.hist) }
    for _, m := range r.hist[len(r.hist)-n:] {
      _ = stream.Send(&chatv1.StreamEnvelope{Payload: &chatv1.StreamEnvelope_Message{Message: m}})
    }
  }
  r.mu.Unlock()

  defer func() { r.mu.Lock(); delete(r.subs, ch); close(ch); r.mu.Unlock() }()

  for {
    select {
    case ev := <-ch:
      if err := stream.Send(ev); err != nil { return err }
    case <-stream.Context().Done():
      return stream.Context().Err()
    }
  }
}

func main() {
  lis, err := net.Listen("tcp", ":50051")
  if err != nil { log.Fatal(err) }
  gs := grpc.NewServer()
  chatv1.RegisterChatServiceServer(gs, &chatServer{h: newHub()})
  log.Println("Go gRPC server listening on :50051")
  if err := gs.Serve(lis); err != nil { log.Fatal(err) }
}

