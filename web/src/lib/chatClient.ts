import { createClient } from "@connectrpc/connect";
import { createGrpcWebTransport } from "@connectrpc/connect-web";
import { ChatService } from './gen/chat/v1/chat_pb';

export const transport = createGrpcWebTransport({
  baseUrl: 'http://localhost:8080',
});

export const chatClient = createClient(ChatService, transport);

