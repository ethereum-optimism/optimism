import * as React from "react"

function LearnIcon({ color }) {
  return (
    <svg
      width={26}
      height={21}
      viewBox="0 0 26 21"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M12.666 12.064l.334.167.335-.167 6.509-3.248v8.283l-5.95 2.574a2.25 2.25 0 01-1.787 0L6.157 17.1V8.821l6.509 3.242z"
        stroke={color}
        strokeWidth={1.5}
      />
      <path
        d="M2.02 6.953L13 1.463l10.98 5.49L13 12.443 2.02 6.953z"
        stroke={color}
        strokeWidth={1.5}
      />
      <path
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        d="M23.875 7v4.125"
      />
    </svg>
  )
}

export default LearnIcon
