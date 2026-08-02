import { cn } from "@/lib/utils";

const LOGO_DARK = "/brand/instrumenta-logo.png";
const LOGO_LIGHT = "/brand/instrumenta-logo-light.png";

type BrandLogoProps = {
  width?: number;
  height?: number;
  className?: string;
};

/** White wordmark for dark theme; black wordmark for light. Colored accents unchanged. */
export function BrandLogo({ width = 140, height, className }: BrandLogoProps) {
  return (
    <>
      <img
        src={LOGO_LIGHT}
        alt="Instrumenta"
        width={width}
        height={height}
        className={cn("dark:hidden", className)}
      />
      <img
        src={LOGO_DARK}
        alt=""
        aria-hidden
        width={width}
        height={height}
        className={cn("hidden dark:block", className)}
      />
    </>
  );
}
