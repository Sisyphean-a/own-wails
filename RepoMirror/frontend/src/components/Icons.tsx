interface IconProps {
  className?: string;
}

export function RepoMirrorLogo({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path
        d="M2.25 3.85C2.25 3.243 2.743 2.75 3.35 2.75H5.65C6.257 2.75 6.75 3.243 6.75 3.85V6.15C6.75 6.757 6.257 7.25 5.65 7.25H3.35C2.743 7.25 2.25 6.757 2.25 6.15V3.85Z"
        stroke="currentColor"
        strokeWidth="1.2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M6.75 5H11.25" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
      <path d="M9.75 3.5L11.25 5L9.75 6.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
      <path
        d="M9.25 9.85C9.25 9.243 9.743 8.75 10.35 8.75H12.65C13.257 8.75 13.75 9.243 13.75 9.85V12.15C13.75 12.757 13.257 13.25 12.65 13.25H10.35C9.743 13.25 9.25 12.757 9.25 12.15V9.85Z"
        stroke="currentColor"
        strokeWidth="1.2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M9.25 11H4.75" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
      <path d="M6.25 9.5L4.75 11L6.25 12.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function ArrowSplitIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 10 10" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M1 5H9" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" />
      <path d="M6.5 2.5L9 5L6.5 7.5" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function BranchIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 10 10" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M3 1.5A1.5 1.5 0 1 1 3 4.5A1.5 1.5 0 0 1 3 1.5Z" stroke="currentColor" strokeWidth="1.1" />
      <path d="M3 4V7C3 7.828 3.672 8.5 4.5 8.5H7" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
      <path d="M7 5.5A1.5 1.5 0 1 1 7 8.5A1.5 1.5 0 0 1 7 5.5Z" stroke="currentColor" strokeWidth="1.1" />
    </svg>
  );
}

export function RefreshIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M9.8 5.2A4 4 0 0 0 2.9 3.4" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
      <path d="M2.8 1.8V3.8H4.8" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M2.2 6.8A4 4 0 0 0 9.1 8.6" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
      <path d="M9.2 10.2V8.2H7.2" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function SwapIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 11 11" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M2 3.2H8.2" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
      <path d="M6.6 1.6L8.2 3.2L6.6 4.8" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M9 7.8H2.8" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
      <path d="M4.4 6.2L2.8 7.8L4.4 9.4" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function FolderIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 10 10" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M1.2 2.4H4L4.8 3.2H8.8V7.8H1.2V2.4Z" stroke="currentColor" strokeWidth="1.1" strokeLinejoin="round" />
    </svg>
  );
}

export function SearchIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 11 11" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <circle cx="5" cy="5" r="3.2" stroke="currentColor" strokeWidth="1.1" />
      <path d="M7.6 7.6L9.4 9.4" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
    </svg>
  );
}

export function SyncIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 11 11" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M4.2 2L7.8 5.5L4.2 9" fill="currentColor" />
    </svg>
  );
}

export function CommitIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 11 11" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <circle cx="5.5" cy="5.5" r="1.6" fill="currentColor" />
      <path d="M1.8 5.5H3.9" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
      <path d="M7.1 5.5H9.2" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
    </svg>
  );
}

export function PushIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 11 11" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M5.5 8.8V2.7" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
      <path d="M3.4 4.5L5.5 2.4L7.6 4.5" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M2 8.8H9" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
    </svg>
  );
}

export function WarningIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M6 1.7L10.5 9.9H1.5L6 1.7Z" stroke="currentColor" strokeWidth="1.1" strokeLinejoin="round" />
      <path d="M6 4.2V6.8" stroke="currentColor" strokeWidth="1.1" strokeLinecap="round" />
      <circle cx="6" cy="8.5" r="0.6" fill="currentColor" />
    </svg>
  );
}

export function ClockIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 9 9" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <circle cx="4.5" cy="4.5" r="3.6" stroke="currentColor" strokeWidth="1" />
      <path d="M4.5 2.6V4.8L6 5.7" stroke="currentColor" strokeWidth="1" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function SaveIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 9 9" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <path d="M1.5 1.5H6.6L7.5 2.4V7.5H1.5V1.5Z" stroke="currentColor" strokeWidth="1" strokeLinejoin="round" />
      <path d="M2.6 1.5V3.4H5.8V1.5" stroke="currentColor" strokeWidth="1" />
      <path d="M3 5.2H6" stroke="currentColor" strokeWidth="1" strokeLinecap="round" />
    </svg>
  );
}

export function SettingsIcon({ className = "" }: IconProps) {
  return (
    <svg className={className} viewBox="0 0 10 10" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <circle cx="5" cy="5" r="1.5" stroke="currentColor" strokeWidth="1" />
      <path
        d="M5 1.6V2.3M5 7.7V8.4M8.4 5H7.7M2.3 5H1.6M7.4 2.6L6.9 3.1M3.1 6.9L2.6 7.4M7.4 7.4L6.9 6.9M3.1 3.1L2.6 2.6"
        stroke="currentColor"
        strokeWidth="1"
        strokeLinecap="round"
      />
    </svg>
  );
}
