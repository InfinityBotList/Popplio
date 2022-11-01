let menuTargets = [];

function insertAfter(newNode, existingNode) {
  existingNode.parentNode.insertBefore(newNode, existingNode.nextSibling);
}

window.addEventListener('DOMContentLoaded', (event) => {
  const rapidocEl = document.getElementById('api');
  rapidocEl.addEventListener('spec-loaded', (e) => {
    let shadow = document.getElementById('api').shadowRoot;

    // Inject CSS
    let style = document.createElement('style');
    
    style.innerHTML = `
      .mobile-menu:hover {
        opacity: 0.8 !important;
      }
    `;
    shadow.appendChild(style);

    let mobileNavEl = shadow.querySelectorAll(".nav-bar-h1")
    
    console.log(mobileNavEl);

    if(mobileNavEl.length <= 0) {
      console.error("No navigation elements found");
      alert("ERROR: No navigation elements found. Please report this bug!");
    }

    // Add all mobile navigation elements
    mobileNavEl.forEach(el => {
      let target = el.getAttribute("data-content-id");
      menuTargets.push({
        "target": target,
        "el": el.innerText.replaceAll("\n", "")
      });
    })

    let mobileTagRoutes = shadow.querySelectorAll(".nav-bar-tag")

    if(mobileTagRoutes.length <= 0) {
      console.error("No navigation elements for tags found");
      alert("ERROR: No navigation elements for tags found. Please report this bug!");
    }

    mobileTagRoutes.forEach(el => {
      let target = el.getAttribute("data-content-id");
      menuTargets.push({
        "target": target,
        "el": `${el.innerText.replaceAll("\n", "")} routes`
      });
    })

    // Close button
    menuTargets.push({
        "el": `<span style="display: flex; items-align: center; margin-top: 30px !important"><svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-x"><line x1="18" y1="6" x2="6" y2="18"></line><line x1="6" y1="6" x2="18" y2="18"></line></svg><span style="margin-left: 3px;">Close</span></span>`
    })

    // Add second showMenu always on top of page
    let showMenuBottom = document.createElement("button");
    showMenuBottom.style = "position: fixed; top: 0; right: 3px; border: none; background: none; font-size: 2em; color: white; padding: 0; margin: 0;";
    showMenuBottom.classList.add("mobile-menu")
    showMenuBottom.innerHTML = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-menu"><line x1="3" y1="12" x2="21" y2="12"></line><line x1="3" y1="6" x2="21" y2="6"></line><line x1="3" y1="18" x2="21" y2="18"></line></svg>`;
    showMenuBottom.ariaLabel = "Show Menu";
    showMenuBottom.onclick = onMobileMenuClick;

    // Add the mobile navigation button to the main header
    shadow.appendChild(showMenuBottom);
  });
});

/* This is evil but needed */
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


function onMobileMenuClick() {
  let shadow = document.getElementById('api').shadowRoot;

  // Create simple navbar below the main header
  mobileNavMenu = document.createElement("div");
  mobileNavMenu.style = "margin: 0; padding: 0; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0, 0, 0, 0.8); z-index: 9999; display: none;";

  let mobileNavMenuContent = document.createElement("div");
  mobileNavMenuContent.style = "margin: 0; padding: 0; position: fixed; top: 0; left: 0; width: 100%; height: 100%; display: flex; flex-direction: column; justify-content: center; align-items: center;";

  let mobileNavMenuContentList = document.createElement("ul");
  mobileNavMenuContentList.style = "list-style: none; padding: 0; margin: 0;";

  menuTargets.forEach(el => {
    let mobileNavMenuContentListItem = document.createElement("li");
    mobileNavMenuContentListItem.style = "margin: 0.5em 0;";
    mobileNavMenuContentListItem.classList.add("mobile-menu")

    let mobileNavMenuContentListItemLink = document.createElement("a");
    mobileNavMenuContentListItemLink.style = "color: white; text-decoration: none; font-size: 16px;";
    mobileNavMenuContentListItemLink.href = "javascript:void(0);"
    mobileNavMenuContentListItemLink.innerHTML = el.el;
    mobileNavMenuContentListItemLink.onclick = () => {
      enableScroll()
      if(el.target) {
        document.getElementById('api').scrollTo(el.target);
      }
      shadow.removeChild(mobileNavMenu);
    }

    mobileNavMenuContentListItem.appendChild(mobileNavMenuContentListItemLink);
    mobileNavMenuContentList.appendChild(mobileNavMenuContentListItem);
  });

  console.log(mobileNavMenu)

  mobileNavMenu.style.display = "block";

  mobileNavMenuContent.appendChild(mobileNavMenuContentList);
  mobileNavMenu.appendChild(mobileNavMenuContent);
  shadow.appendChild(mobileNavMenu);

  disableScroll()
}
