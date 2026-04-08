import { initializeApp } from 'firebase/app'
import { getAuth, GoogleAuthProvider, signInWithPopup, signOut, onAuthStateChanged } from 'firebase/auth'

const firebaseConfig = {
  apiKey: "AIzaSyBFgf1nFyEKZptLv66IwYCRoBlm5a1BrFg",
  authDomain: "fin-cascade.firebaseapp.com",
  projectId: "fin-cascade",
  storageBucket: "fin-cascade.firebasestorage.app",
  messagingSenderId: "259197738252",
  appId: "1:259197738252:web:57ac840fcfeff14ae7bd29",
}

const app = initializeApp(firebaseConfig)
const auth = getAuth(app)
const googleProvider = new GoogleAuthProvider()

export async function signInWithGoogle() {
  const result = await signInWithPopup(auth, googleProvider)
  return result.user
}

export async function logOut() {
  await signOut(auth)
}

export function onAuth(callback) {
  return onAuthStateChanged(auth, callback)
}

export async function getIdToken() {
  const user = auth.currentUser
  if (!user) return null
  return user.getIdToken()
}

export { auth }
