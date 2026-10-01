import React from 'react';

function Icon({ children, size = 16, ...props }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      {...props}
    >
      {children}
    </svg>
  );
}

export const SidebarIcon = (props) => (
  <Icon {...props}>
    <rect x="3" y="4.5" width="18" height="15" rx="2" />
    <path d="M9 4.5v15" />
  </Icon>
);

export const RowsIcon = (props) => (
  <Icon {...props}>
    <rect x="4" y="4" width="16" height="7" rx="1.5" />
    <rect x="4" y="13" width="16" height="7" rx="1.5" />
  </Icon>
);

export const ColumnsIcon = (props) => (
  <Icon {...props}>
    <rect x="4" y="4" width="7" height="16" rx="1.5" />
    <rect x="13" y="4" width="7" height="16" rx="1.5" />
  </Icon>
);

export const WaveformIcon = (props) => (
  <Icon {...props}>
    <path d="M2 12h3l2-6 3 12 3-15 3 15 2-6h4" />
  </Icon>
);

export const AlertIcon = (props) => (
  <Icon {...props}>
    <circle cx="12" cy="12" r="9" />
    <path d="M12 7.5v5.5" />
    <path d="M12 16.5h.01" />
  </Icon>
);
