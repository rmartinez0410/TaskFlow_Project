import { Controller, Get, Post, Body, Patch, Param, Delete, UseGuards, Inject } from '@nestjs/common';
import { Registeruserdto } from './dto/register-user.dto';
import { LoginUserDto } from './dto/login-user.dto';
import { AuthGuard } from './auth.guard';
import { NATS_SERVICES } from 'src/config/services';
import { ClientProxy, RpcException } from '@nestjs/microservices';
import { catchError } from 'rxjs';
import { User } from './decorators/user.decorator';


@Controller('auth')
export class AuthController {
  constructor(@Inject(NATS_SERVICES) private readonly client : ClientProxy) {}

  @Post('register')
  register (@Body() registerUserDto: Registeruserdto) {
    return this.client.send('auth.register', registerUserDto)
     .pipe(
      catchError((err) => {
        throw new RpcException(err.message);
      })
    );
  }


  @Post('login')
  login(@Body() loginUserDto: LoginUserDto){
    return this.client.send('auth.login', loginUserDto).pipe(
      catchError((err) => {
        throw new RpcException(err.message);
      })
    );
  }

  @UseGuards(AuthGuard)
  @Get()
  verify(@User() user: any){
    return user ;
  }

  
}
