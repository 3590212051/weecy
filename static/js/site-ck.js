/**
 *  @create 2013/06/11
 *  @version 0.2
 *  @author: chenwenli <kapa2robert@gmail.com>, Unknown <joe2010xtmf@163.com>
 */function AddLabelSubmit(e){var t=$.trim(document.getElementById("label_box").value);if(t.length>0){var n;window.location.href.indexOf("?")>-1?n=window.location.href.replace("?",":l="+t+"?"):n=window.location.href+":l="+t;window.location.href=n;return!1}}function RemoveLabelSubmit(e){var t=$.trim(document.getElementById("label_box").value);if(t.length>0){var n;window.location.href.indexOf("?")>-1?n=window.location.href.replace("?",":rl="+t+"?"):n=window.location.href+":rl="+t;window.location.href=n;return!1}}function showExample(e){var t=document.getElementById("_ex_"+e);t.className="accordion-body collapse in";t.style.height="auto"}function createXMLHttpRequest(){window.ActiveXObject?xmlHttp=new ActiveXObject("Microsoft.XMLHTTP"):window.XMLHttpRequest&&(xmlHttp=new XMLHttpRequest)}function getFuncCode(e){var t=document.getElementById("pid"),n=document.getElementById("collapse_"+e),r=n.getElementsByTagName("pre");if(r[0].innerHTML!=="LOADING...")return;createXMLHttpRequest();xmlHttp||(r[0].innerHTML="Fail to create XMLHTTP.");funcName=e;xmlHttp.open("GET","/funcs?name="+e+"&pid="+t.innerHTML,!0);xmlHttp.onreadystatechange=okFunc;xmlHttp.setRequestHeader("Content-Type","application/x-www-form-urlencoded");xmlHttp.send()}(function(){function e(){var e=document.getElementById("navbar_frame"),t=document.getElementById("top_search_form"),n=document.getElementById("navbar_search_box"),r=document.getElementById("body"),i=document.getElementById("main_content"),s=document.body.clientWidth-1100;if(s>0){e.style.width="";if(document.getElementById("sidebar")==null){e.className="navbar navbar-fixed-top";r.style.paddingTop="60px"}else{e.className="navbar";r.style.paddingTop="0px"}t.className="navbar-search pull-right";n.style.width="";i!=null&&(i.style.marginLeft="-20px")}else{e.style.width="1000px";e.className="navbar";t.className="navbar-search";n.style.width="150px";r.style.paddingTop="0px";i!=null&&(i.style.marginLeft="0px")}}function l(e){f==1&&e();f=0}e();$(window).resize(function(){e()});var t="Back to Top",n=$('<div class="backToTop"></div>').appendTo($("body")).attr("title",t).click(function(){$("html, body").animate({scrollTop:0},120)}),r=function(){var e=$(document).scrollTop(),t=$(window).height();e>0?n.show():n.hide();window.XMLHttpRequest||n.css("top",e+t-166)};$(window).bind("scroll",r);r();document.body.clientWidth>1500&&document.getElementById("sidebar")!=null&&(document.getElementById("sidebar").className="span3");var i=$("#search_exports");if(i.length!=0){i.modal({keyboard:!1,show:!1});$("#search_form").submit(function(){var e=$.trim(document.getElementById("search_export_box").value);if(e.length>0){i.modal("hide");var t="#".concat(e.replace(".","_"));location.hash=t}i.find("input[type=text]").val("");return!1})}else i=null;var s=$("#label_modal");if(s.length!=0){s.modal({keyboard:!1,show:!1});$("#label_form").submit(function(){var e=$("#label_modal");e.modal("hide");e.find("input[type=text]").val("");return!1})}else s=null;var o=$("#example_modal");o.length==0&&(o=null);var u=$("#_keyshortcut");u.modal({keyboard:!0,show:!1});var a=0,f=0;document.getElementById("sidebar")!=null?a=1:u.find("tbody > tr").each(function(e,t){(e==2||e==5||e==6||e==7)&&$(t).addClass("muted")});$(document).keypress(function(e){if($("input:focus").length!=0)return!0;var t=e.keyCode?e.keyCode:e.charCode;if(t==63){i&&i.modal("hide");s&&s.modal("hide");o&&o.modal("hide");u.modal("show")}else{if(t==47){i&&i.modal("hide");s&&s.modal("hide");o&&o.modal("hide");u.modal("hide");$("input[name=q]").first().focus();return!1}if(t==46&&a){s&&s.modal("hide");o&&o.modal("hide");u.modal("hide");if(i){i.modal("show");i.on("shown",function(){$(this).find("#search_export_box").focus()})}}else if(t==103){i&&i.modal("hide");s&&s.modal("hide");o&&o.modal("hide");u.modal("hide");if(f==0){f=1;setTimeout(function(){f=0},2e3);return!1}l(function(){$("html,body").animate({scrollTop:0},120)})}else if(t==98){i&&i.modal("hide");s&&s.modal("hide");o&&o.modal("hide");u.modal("hide");l(function(){$("html,body").animate({scrollTop:$("body").height()},120)})}else if(t==105){i&&i.modal("hide");s&&s.modal("hide");o&&o.modal("hide");u.modal("hide");l(function(){location.hash="#_index"})}else if(t==116){i&&i.modal("hide");o&&o.modal("hide");u.modal("hide");if(s){s.modal("show");s.on("shown",function(){$(this).find("#label_box").focus()})}}else if(t==101){i&&i.modal("hide");s&&s.modal("hide");u.modal("hide");if(o){o.modal("show");o.on("shown",function(){$(this).find("#example_box").focus()})}}}})})();var xmlHttp,funcName,okFunc=function(){if(xmlHttp.readyState==4&&xmlHttp.status==200){var e=document.getElementById("collapse_"+funcName),t=e.getElementsByTagName("pre");t[0].innerHTML=xmlHttp.responseText}};