export default function Footer() {
  return (
    <footer className="footer">
      <div className="container">
        <div className="footer-inner">
          <div className="footer-brand">
            <img src="/logo-kong.png" alt="Kongtrol" />
            <span className="footer-brand-name">VPN KONGTROL</span>
          </div>

          <ul className="footer-links">
            <li><a href="#descargar">Descargar</a></li>
            <li><a href="#guia">Guía</a></li>
          </ul>

          <span className="footer-copy">MIT License</span>
        </div>
      </div>
    </footer>
  )
}
