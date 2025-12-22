import { IsEmail, IsString, MaxLength, MinLength } from "class-validator";

export class Registeruserdto{
    
    @IsString()
    @MinLength(2)
    @MaxLength(30)
    username: String;
    
    @IsEmail()
    @IsString()
    email: String;
    
    @IsString()
    @MinLength(6)
    @MaxLength(50)
    password: String;
}