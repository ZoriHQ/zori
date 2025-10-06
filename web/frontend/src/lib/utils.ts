import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'
import type { ClassValue } from 'clsx'

export function cn(...inputs: Array<ClassValue>) {
  return twMerge(clsx(inputs))
}

function extractMessage(str: string): string {
  const messageIndex = str.indexOf('message=')

  if (messageIndex === -1) {
    return str
  }

  const messageStart = messageIndex + 'message='.length
  let message = str.substring(messageStart)

  const commaIndex = message.indexOf(',')
  if (commaIndex !== -1) {
    message = message.substring(0, commaIndex)
  }

  return message.trim()
}

export function formatApiError(err: any): string {
  if (typeof err === 'object') {
    if (err.message) {
      const messageParts = err.message.includes('code=')
        ? err.message.split(',')
        : [err.message]

      if (messageParts.length === 2) {
        const [, message] = messageParts
        return `${message.replace('message=', '')}`
      }

      return err.message
    }
    return JSON.stringify(err)
  }
  try {
    const parsedErr = JSON.parse(err)
    if (parsedErr.error) {
      return extractMessage(parsedErr.error)
    }
  } catch (error) {
    console.error('Error parsing JSON:', error)
  }

  return String(err)
}
