import * as React from "react"

function WalletIcon({ color }) {
  return (
    <svg
      width={18}
      height={15}
      viewBox="0 0 18 15"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      // {...props}
    >
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M15 1.5H1.5v11.625H15a1.5 1.5 0 001.5-1.5V3A1.5 1.5 0 0015 1.5zM1.5 0H0v14.625h15a3 3 0 003-3V3a3 3 0 00-3-3H1.5zm13.125 4.125a.75.75 0 01-.75.75H9.75a.75.75 0 010-1.5h4.125a.75.75 0 01.75.75zm-.75 4.125a.75.75 0 000-1.5H12a.75.75 0 000 1.5h1.875z"
        fill={color}
      />
    </svg>
  )
}

export default WalletIcon
