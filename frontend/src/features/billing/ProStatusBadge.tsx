import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { proBadgeAriaLabel, proBadgeText } from "@/features/billing/proStatus";

type Props = {
  endAt?: string | null;
  complimentary?: boolean;
  /** Short «Pro» for bottom nav. */
  compact?: boolean;
  className?: string;
  /** When false, render plain badge without link (e.g. already on plan page). */
  linkToPlan?: boolean;
};

export function ProStatusBadge({
  endAt,
  complimentary = false,
  compact = false,
  className,
  linkToPlan = true,
}: Props) {
  const label = proBadgeText({ endAt, complimentary, compact });
  const aria = proBadgeAriaLabel({ endAt, complimentary });

  const badge = (
    <Badge
      variant="secondary"
      className={cn(
        "border-transparent bg-primary/15 font-semibold text-primary",
        compact ? "h-4 min-w-4 px-1 text-[10px]" : "text-xs font-medium",
        className,
      )}
    >
      {label}
    </Badge>
  );

  if (!linkToPlan) {
    return (
      <span aria-label={aria} title={aria}>
        {badge}
      </span>
    );
  }

  return (
    <Link
      to="/account"
      className="inline-flex min-h-10 items-center"
      aria-label={aria}
      title={aria}
    >
      {badge}
    </Link>
  );
}
