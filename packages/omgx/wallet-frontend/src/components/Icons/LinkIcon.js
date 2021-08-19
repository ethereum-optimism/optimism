import * as React from "react";

function LinkIcon({color = 'white'}) {
    
    return (<svg
        width="32"
        height="32"
        viewBox="0 0 32 32"
        fill="none"
        xmlns="http://www.w3.org/2000/svg">
        <path
            d="M14.7249 11.4734L14.5004 11.3925C13.7712 11.1294 12.9557 11.3114 12.4075 11.8596L7.24833 17.0188C6.07676 18.1903 6.07676 20.0898 7.24833 21.2614L8.31797 22.331C9.48954 23.5026 11.389 23.5026 12.5606 22.331L17.6051 17.2866C18.2107 16.681 18.3635 15.757 17.9851 14.9887L17.9676 14.9532"
            stroke={color}
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
        />
        <path
            d="M14.239 17.2252L14.1154 16.7466C13.9385 16.061 14.1371 15.3331 14.6378 14.8325L19.6212 9.84903C20.7928 8.67746 22.6922 8.67746 23.8638 9.84903L24.7518 10.737C25.9234 11.9086 25.9234 13.8081 24.7518 14.9797L19.5985 20.133C19.1074 20.6241 18.3967 20.8253 17.7211 20.6645L17.4641 20.6033"
            stroke={color}
            strokeWidth="1.5"
            strokeLinecap="round"
            strokeLinejoin="round"
        />
    </svg>)
}

export default LinkIcon;
