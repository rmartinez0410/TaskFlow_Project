import { IsEmail, IsString, MaxLength, MinLength } from 'class-validator';

export class Registeruserdto {
  @IsEmail()
  @IsString()
  email: String;

  @IsString()
  @MinLength(6)
  @MaxLength(50)
  password: String;

  @IsString()
  @MinLength(2)
  @MaxLength(30)
  username: String;
}
