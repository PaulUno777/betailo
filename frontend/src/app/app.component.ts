import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  templateUrl: './app.component.html',
  styleUrl: './app.component.css'
})
export class AppComponent {
  // Nom affiché dans le bandeau d'amorçage ci-dessous — à remplacer par la
  // vraie mise en page une fois les maquettes (voir document de proposition,
  // section "Direction artistique") disponibles.
  title = 'Betailo';
}
