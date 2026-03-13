import { IsString, MaxLength, MinLength } from "class-validator";

export class refreshTokenDTO {
  @IsString()
  @MinLength(6)
  @MaxLength(900)
  refresh_token: string;
}
