/**
 * Root Page - Redirects to Operations
 * Operations is the main working area of ISX Pulse
 */

import { redirect } from 'next/navigation'

export default function HomePage() {
  // Redirect to operations - the main working area
  redirect('/operations')
}