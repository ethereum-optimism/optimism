import * as React from "react"

function SearchIcon({ color }) {
    return (
        <svg
            width="23"
            height="23"
            viewBox="0 0 23 23"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
        >
            <line
                x1="14.2154"
                y1="14.3477"
                x2="17.1548"
                y2="17.287"
                stroke={color}
                strokeWidth="1.5"
                strokeLinecap="round"
            />
            <circle
                cx="9.50958"
                cy="9.83984"
                r="5.97429"
                transform="rotate(-45 9.50958 9.83984)"
                stroke={color}
                strokeWidth="1.5"
            />
        </svg>

    )
}

export default SearchIcon
