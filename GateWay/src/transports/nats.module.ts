import { Module } from "@nestjs/common";
import { ClientsModule, Transport } from "@nestjs/microservices";
import { Server } from "http";
import { options } from "joi";
import { enviromenmtsVariable } from "src/config";
import { NATS_SERVICES } from "src/config/services";

@Module({
    imports: [
        ClientsModule.register([
            {
                name: NATS_SERVICES,
                transport: Transport.NATS,
                options: {
                    servers: enviromenmtsVariable.nastServer,
                }
            }
        ])
    ],
    exports:[
        ClientsModule.register([
            {
                name: NATS_SERVICES,
                transport: Transport.NATS,
                options: {
                    servers: enviromenmtsVariable.nastServer,
                }
            }
        ])
    ]
})
export class NatsModule {}