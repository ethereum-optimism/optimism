import * as React from "react"

function EarnIcon({ color }) {
  return (
    <svg
      width={22}
      height={17}
      viewBox="0 0 22 17"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M.875 10.438l6.75-5.626 3.938 3.938 9-7.875M2 15.5h17.438"
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
      />
    </svg>
  )
}

export default EarnIcon
