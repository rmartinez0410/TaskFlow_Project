import { NestFactory } from '@nestjs/core';
import { AppModule } from './app.module';
import { Logger, ValidationPipe } from '@nestjs/common';
import { enviromenmtsVariable } from './config';
import { RpcExceptionsFilter } from './common/exceptions/rpc-exceptions.filter';


async function main() {
  const logger = new Logger("Gateway");
  const app = await NestFactory.create(AppModule);
  app.setGlobalPrefix("api")
  app.useGlobalFilters(
    new RpcExceptionsFilter()
  );
  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      forbidNonWhitelisted: true,
    })
  )
  await app.listen(enviromenmtsVariable.port ?? 3000);
  logger.log(`Gateway is running in port `,enviromenmtsVariable.port)
}
main();
