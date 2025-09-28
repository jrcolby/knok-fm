interface KnokStarProps {
  className?: string;
}

export function KnokStar({ className }: KnokStarProps) {
  return (
    <svg
      width="481"
      height="417"
      viewBox="0 0 481 417"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      <path
        d="M202.56 229.85L120.13 416.16L240.27 251.62L360.4 416.16L277.98 229.85L480.53 208.08L277.98 186.31L360.4 0L240.27 164.54L120.13 0L202.56 186.31L0 208.08L202.56 229.85Z"
        fill="currentColor"
      />
    </svg>
  );
}
