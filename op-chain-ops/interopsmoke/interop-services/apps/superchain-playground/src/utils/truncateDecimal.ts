export const truncateDecimal = (
  decimalString: string,
  maxLengthAfterDecimal = 5,
): string => {
  if (decimalString.includes('.')) {
    const parts = decimalString.split('.')
    const integerPart = parts[0]!
    const decimalPart = parts[1]!
      .substring(0, maxLengthAfterDecimal)
      .replace(/0+$/, '')

    return decimalPart ? `${integerPart}.${decimalPart}` : integerPart
  }

  return decimalString
}
