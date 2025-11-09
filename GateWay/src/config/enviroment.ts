import 'dotenv/config';
import * as joi from 'joi';

interface EviromentVariables{
    PORT: number;
    NATS_SERVER: string;
}

const enviromentSchema = joi.object({
    PORT: joi.number().required(),
    NATS_SERVER: joi.string().required(),
}).unknown();

const {error , value} = enviromentSchema.validate({
    ... process.env
});

if(error){
    throw new Error(`EnviromentError ${error.message}`)
}

const env: EviromentVariables = value;
export const enviromenmtsVariable = {
    port: env.PORT,
    nastServer: env.NATS_SERVER
}
