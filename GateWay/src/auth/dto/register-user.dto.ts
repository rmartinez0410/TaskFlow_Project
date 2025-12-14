import { IsEmail, IsString, MaxLength, MinLength } from "class-validator";

export class Registeruserdto{
    
    @IsString()
    name: String;
    
    @IsEmail()
    @IsString()
    email: String;
    
    @IsString()
    @MinLength(6)
    @MaxLength(50)
    password: String;
}