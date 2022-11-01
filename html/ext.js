/*
Below is the expanded implementation of disableScroll() and enableScroll() functions as used in the below code.

function disableScroll() {
  // Get the current page scroll position
  scrollTop = window.pageYOffset || document.documentElement.scrollTop;
  scrollLeft = window.pageXOffset || document.documentElement.scrollLeft,

  // if any scroll is attempted, set this to the previous value
  window.onscroll = function() {
      window.scrollTo(scrollLeft, scrollTop);
  };
}
  
function enableScroll() {
    window.onscroll = function() {};
}
*/

window.addEventListener('DOMContentLoaded', () => {
  const rapidocEl = document.getElementById('api');
  rapidocEl.addEventListener('spec-loaded', () => {
    let menuTargets = [];

    let shadow = document.getElementById('api').shadowRoot;

    // Inject CSS
    let style = document.createElement('style');
    
    style.innerHTML = `.mobile-menu:hover {opacity:0.8!important;}`;
    shadow.appendChild(style);

    // Add all mobile navigation elements
    shadow.querySelectorAll(".nav-bar-h1").forEach(el => {
      menuTargets.push({
        "target": el.getAttribute("data-content-id"),
        "el": el.innerText.replaceAll("\n", "")
      });
    })

    shadow.querySelectorAll(".nav-bar-tag").forEach(el => {
      menuTargets.push({
        "target": el.getAttribute("data-content-id"),
        "el": `${el.innerText.replaceAll("\n", "")} routes`
      });
    })

    // Close button
    menuTargets.push({
        "el": `<span style="display:flex;items-align:center;margin-top:30px!important"><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg><span style="margin-left:3px;">Close</span></span>`
    })

    // Add second showMenu always on top of page
    let showMenuBottom = document.createElement("button");
    showMenuBottom.style = "position:fixed;top:0;right:3px;border:none;background:none;font-size:2em;color:white;";
    showMenuBottom.classList.add("mobile-menu")
    showMenuBottom.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="3" y1="12" x2="21" y2="12"></line><line x1="3" y1="6" x2="21" y2="6"></line><line x1="3" y1="18" x2="21" y2="18"></line></svg>`;
    showMenuBottom.ariaLabel = "Show Menu";
    showMenuBottom.onclick = () => {
        // Common styles
        let fullScreen = "margin:0;padding:0;position:fixed;top:0;left:0;width:100%;height:100%;"

        // Create simple navbar below the main header
        menu = document.createElement("div");
        menu.style = fullScreen+"background:rgba(0, 0, 0, 0.8);z-index:9999;display:none;";

        let menuContent = document.createElement("div");
        menuContent.style = fullScreen+"display:flex;flex-direction:column;justify-content:center;align-items:center";

        let menuContentList = document.createElement("ul");
        menuContentList.style = "list-style:none;padding:0;margin:0;";

        menuTargets.forEach(el => {
            let menuContentListItem = document.createElement("li");
            menuContentListItem.style = "margin: 0.5em 0;";
            menuContentListItem.classList.add("mobile-menu")

            let menuContentListItemLink = document.createElement("a");
            menuContentListItemLink.style = "color:white;text-decoration:none;font-size:16px";
            menuContentListItemLink.href = "javascript:void(0);"
            menuContentListItemLink.innerHTML = el.el;
            menuContentListItemLink.onclick = () => {
            window.onscroll = function() {}; // enableScroll();
            if(el.target) {
                document.getElementById('api').scrollTo(el.target);
            }
            shadow.removeChild(menu);
            }

            menuContentListItem.appendChild(menuContentListItemLink);
            menuContentList.appendChild(menuContentListItem);
        });

        //console.log(menu)

        menu.style.display = "block";

        menuContent.appendChild(menuContentList);
        menu.appendChild(menuContent);
        shadow.appendChild(menu);

        // disableScroll() implementation
        let scrollTop = window.scrollY;
        let scrollLeft = window.scrollX;
        window.onscroll = function() {
            window.scrollTo(scrollLeft, scrollTop);
        };
    };

    // Add the mobile navigation button to the main header
    shadow.appendChild(showMenuBottom);
  });
});
