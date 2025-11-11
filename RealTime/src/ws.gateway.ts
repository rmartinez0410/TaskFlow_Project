// realtime/ws.gateway.ts
import {
  WebSocketGateway,
  WebSocketServer,
  SubscribeMessage,
  MessageBody,
  ConnectedSocket,
  OnGatewayConnection,
  OnGatewayDisconnect,
} from '@nestjs/websockets';
import { Server, Socket } from 'socket.io';
import { EventPattern } from '@nestjs/microservices';

@WebSocketGateway({ namespace: '/ws', cors: true })
export class WsGateway implements OnGatewayConnection, OnGatewayDisconnect {
  @WebSocketServer()
  server: Server;

  handleConnection(client: Socket) {
    const userId = client.handshake.auth?.userId;
    if (userId) {
      client.join(`user:${userId}`);
      console.log(`Usuario ${userId} conectado`);
    }
  }

  handleDisconnect(client: Socket) {
    console.log(`Cliente desconectado: ${client.id}`);
  }

  @SubscribeMessage('subscribe_user')
  handleSubscribe(@MessageBody() data: { userId: string }, @ConnectedSocket() client: Socket) {
    client.join(`user:${data.userId}`);
    return { ok: true };
  }

  @EventPattern('notificacion.creada')
  handleNotificacion(data: { userId: string; mensaje: string }) {
    this.server.to(`user:${data.userId}`).emit('notificacion', data);
  }

  @EventPattern('tarea.asignada')
  handleTareaAsignada(data: { userId: string; tareaId: string; titulo: string }) {
    this.server.to(`user:${data.userId}`).emit('tarea', data);
  }
}
