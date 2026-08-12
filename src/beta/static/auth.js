// Minimal client-side UX: intercept form submits to show simple feedback
document.addEventListener('submit', async function(e){
  const form = e.target;
  if(form.id === 'loginForm' || form.id === 'registerForm'){
    e.preventDefault();
    const data = new URLSearchParams(new FormData(form));
    const resp = await fetch(form.action, {method: 'POST', body: data});
    if(resp.ok){
      // on login, go to root
      if(form.id === 'loginForm'){
        window.location = '/';
      } else {
        alert('Account created. You can now sign in.');
        window.location = '/login';
      }
    } else {
      const txt = await resp.text();
      alert('Error: ' + txt);
    }
  }
});
