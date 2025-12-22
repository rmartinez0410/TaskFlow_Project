import {
  CanActivate,
  ExecutionContext,
  Inject,
  UnauthorizedException,
} from '@nestjs/common';
import { firstValueFrom, Observable } from 'rxjs';
import { Request } from 'express';
import { NATS_SERVICES } from 'src/config';
import { ClientProxy } from '@nestjs/microservices';
import { User } from './entities/auth.entity';

export class AuthGuard implements CanActivate {
  constructor(@Inject(NATS_SERVICES) private readonly client: ClientProxy) {}

  async canActivate(context: ExecutionContext): Promise<boolean> {
    const request = context.switchToHttp().getRequest();
    const token = this.tokenExtractor(request);

    if (!token) {
      throw new UnauthorizedException();
    }

    try {
      const user: User = await firstValueFrom(
        this.client.send('auth.validate', token),
      );
      request.user = user;
    
      return true;
    } catch (error: any) {
      if (error.response) {
        const { statusCode, data } = error.response;

        if (data == true) {
          throw new UnauthorizedException(error);
        } else {
          const newToken = await this.client.send('auth.refresh', token);

          request.token = newToken;

          return true;
        }
      }

      return false;
    }
  }

  tokenExtractor(request: Request): string | undefined {
    const [type, token] = request.headers.authorization?.split(' ') ?? [];

    if (type === 'Bearer' && token != ' ') {
      return token;
    }

    return undefined;
  }
}
