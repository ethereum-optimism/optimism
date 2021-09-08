import * as React from "react"

function PoolIcon({ color }) {
  return (
    <svg
      width={16}
      height={20}
      viewBox="0 0 16 20"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M15.125 12a7.125 7.125 0 01-14.25 0c0-3.126 1.568-5.91 3.332-7.946a15.901 15.901 0 012.531-2.365c.374-.274.697-.476.946-.605A2.296 2.296 0 018 .945a2.296 2.296 0 01.316.139c.25.13.572.33.946.605a15.901 15.901 0 012.53 2.365c1.765 2.036 3.333 4.82 3.333 7.946z"
        stroke={color}
        strokeWidth={1.5}
      />
      <path
        stroke={color}
        strokeWidth={1.5}
        strokeLinecap="round"
        d="M10.386 9.869l-4.507 4.508M10.121 14.379L5.613 9.871"
      />
    </svg>
  )
}

export default PoolIcon
