import { ArgumentsHost, Catch, ExceptionFilter } from "@nestjs/common";
import { RpcException } from "@nestjs/microservices";

@Catch(RpcException)
export class RpcExceptionsFilter implements ExceptionFilter {
  catch(exception: RpcException, host: ArgumentsHost) {
    const ctx = host.switchToHttp();
    const response = ctx.getResponse();

    const rpcError = exception.getError();

    if(rpcError.toString().includes('Empty response')){
        return response.status(500).json({
            statusCode: 500,
            message: rpcError
            .toString()
            .substring(0, rpcError.toString().indexOf('(') -1 ),
        });
    }

    function isRpcError(obj: unknown): obj is { status: number | string, message: string } {
  return typeof obj === 'object' &&
         obj !== null &&
         'status' in obj &&
         'message' in obj;
}

if (isRpcError(rpcError)) {
    const httpStatus = isNaN(+rpcError.status) ? 400 : +rpcError.status;
    return response.status(httpStatus).json(rpcError);
}
    return response.status(400).json({
        statusCode: 400,
        message: rpcError.toString(),
    }); 
  }
}