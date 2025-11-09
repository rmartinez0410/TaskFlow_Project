import { CanActivate, ExecutionContext, Inject, UnauthorizedException } from "@nestjs/common";
import { firstValueFrom, Observable } from "rxjs";
import { Request } from 'express';
import { NATS_SERVICES } from "src/config";
import { ClientProxy } from "@nestjs/microservices";

export class AuthGuard implements CanActivate{

    constructor(@Inject(NATS_SERVICES) private readonly client: ClientProxy ){}

    async canActivate(context: ExecutionContext): Promise<boolean> {
        const request = context.switchToHttp().getRequest();
        const token = this.tokenExtractor(request);

        if(!token){
            throw new UnauthorizedException();
        }

        try {
            const {user, token: newToken }= await firstValueFrom(
                this.client.send('verify.token', token)
            )
            request.user = user ;
            request.token = newToken ;
            return true;
        } catch (error) {
            throw new UnauthorizedException(error);
        }
    }

    tokenExtractor(request: Request): string | undefined {
        const [type, token]  = request.headers.authorization?.split(" ") ?? [];
    
        if( type === 'Bearer' && token != " "){
        return token
        }

        return undefined;
    }
}