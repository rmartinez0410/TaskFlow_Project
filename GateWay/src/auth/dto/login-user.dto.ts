import { IsBoolean, IsEmail, IsString, MaxLength, MinLength } from 'class-validator';

export class LoginUserDto {
  @IsEmail()
  @IsString()
  email: string;

  @IsString()
  @MinLength(6)
  @MaxLength(50)
  password: string;

  @IsString()
  @MinLength(6)
  @MaxLength(50)
  device_name: string;
  
  @IsString()
  @MinLength(6)
  @MaxLength(50)
  device_type: string;

  @IsBoolean()
  remember_me: boolean;  
  
  @IsString()
  @MinLength(6)
  @MaxLength(50)
  ip_address: string;
 
  @IsString()
  @MinLength(6)
  @MaxLength(50)
  user_agent: string;

}
