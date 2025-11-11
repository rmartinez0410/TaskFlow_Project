import { IsEmail, IsString, MaxLength, MinLength } from "class-validator";

export class Registeruserdto{
    
    @IsString()
    name: string;
    
    @IsEmail()
    @IsString()
    email: string;
    
    @IsString()
    @MinLength(6)
    @MaxLength(50)
    password: string;
}